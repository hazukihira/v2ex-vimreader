package service

import (
	"context"
	"strings"
	"time"
)

const (
	cacheRetentionDays = 3
	cacheArchiveDays   = 30
	cacheAuxRetention  = 30 * 24 * time.Hour
	cachePrefetchBatch = 200
	cacheMaintInterval = 5 * time.Minute
	cacheInitialDelay  = 20 * time.Second
	cachePruneMetaKey  = "cache.cleanup.day.v2"
)

func (s *Service) cacheMaintenanceLoop(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(cacheInitialDelay):
	}
	s.cacheMaintenance(ctx)
	ticker := time.NewTicker(cacheMaintInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cacheMaintenance(ctx)
		}
	}
}

func (s *Service) cacheMaintenance(ctx context.Context) {
	s.pruneOldCache(ctx)
	s.prefetchToday(ctx)
}

func (s *Service) localDayStart(t time.Time) time.Time {
	local := t.In(s.loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, s.loc)
}

func (s *Service) pruneOldCache(ctx context.Context) {
	start := s.localDayStart(time.Now())
	if s.metaInt(ctx, cachePruneMetaKey) == start.Unix() {
		return
	}
	contentCutoff := start.AddDate(0, 0, -(cacheRetentionDays - 1))
	removed, err := s.st.PurgeTopicCacheBefore(ctx, contentCutoff.Unix())
	if err != nil {
		s.log.Warn("topic cache purge failed", "err", err)
		return
	}
	archiveCutoff := start.AddDate(0, 0, -(cacheArchiveDays - 1))
	archived, err := s.st.PurgeArchiveBefore(ctx, archiveCutoff.Unix())
	if err != nil {
		s.log.Warn("archive purge failed", "err", err)
		return
	}
	summaryCutoff := time.Now().Add(-summaryTTL).Unix()
	summaries, err := s.st.PurgeSummariesBefore(ctx, summaryCutoff)
	if err != nil {
		s.log.Warn("summary cache purge failed", "err", err)
		return
	}
	auxCutoff := time.Now().Add(-cacheAuxRetention).Unix()
	nodes, err := s.st.PurgeNodesBefore(ctx, auxCutoff)
	if err != nil {
		s.log.Warn("node cache purge failed", "err", err)
		return
	}
	feeds, err := s.st.PurgeFeedStatesBefore(ctx, auxCutoff)
	if err != nil {
		s.log.Warn("feed state purge failed", "err", err)
		return
	}
	s.metaSetInt(ctx, cachePruneMetaKey, start.Unix())
	if removed > 0 || archived > 0 || summaries > 0 || nodes > 0 || feeds > 0 {
		s.log.Info("cache purged",
			"topic_bodies", removed, "archive_topics", archived, "summaries", summaries,
			"nodes", nodes, "feed_states", feeds,
			"content_cutoff", contentCutoff.Format("2006-01-02"),
			"archive_cutoff", archiveCutoff.Format("2006-01-02"))
	}
}

func (s *Service) prefetchToday(ctx context.Context) {
	start := s.localDayStart(time.Now())
	ids, err := s.st.TopicIDsForWarmup(ctx, start.Unix(), start.Add(24*time.Hour).Unix(), cachePrefetchBatch)
	if err != nil {
		s.log.Warn("list today cache candidates failed", "err", err)
		return
	}
	if len(ids) == 0 {
		return
	}
	fetched, failed := 0, 0
	now := time.Now()
	s.prefetchMu.Lock()
	for id, at := range s.prefetchFailed {
		if now.Sub(at) > 24*time.Hour {
			delete(s.prefetchFailed, id)
		}
	}
	s.prefetchMu.Unlock()
	for _, id := range ids {
		select {
		case <-ctx.Done():
			return
		default:
		}
		s.prefetchMu.Lock()
		retryAt, retryBlocked := s.prefetchFailed[id]
		s.prefetchMu.Unlock()
		if retryBlocked && now.Before(retryAt.Add(time.Hour)) {
			continue
		}
		topic, err := s.st.GetTopic(ctx, id)
		if err != nil {
			failed++
			continue
		}
		if _, restricted := s.restricted[strings.ToLower(topic.Node)]; restricted {
			continue
		}
		if _, err := s.fetchTopicPage(ctx, id, 1); err != nil {
			failed++
			s.prefetchMu.Lock()
			if s.prefetchFailed == nil {
				s.prefetchFailed = make(map[int64]time.Time)
			}
			s.prefetchFailed[id] = now
			s.prefetchMu.Unlock()
			s.log.Warn("prefetch topic failed", "id", id, "err", err)
			continue
		}
		s.prefetchMu.Lock()
		if s.prefetchFailed != nil {
			delete(s.prefetchFailed, id)
		}
		s.prefetchMu.Unlock()
		fetched++
	}
	if fetched > 0 || failed > 0 {
		s.log.Info("today topic cache prefetched", "fetched", fetched, "failed", failed, "total", len(ids))
	}
}
