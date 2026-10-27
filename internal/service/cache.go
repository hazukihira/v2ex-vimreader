package service

import (
	"context"
	"strings"
	"time"
)

const (
	cacheRetentionDays = 3
	cachePrefetchBatch = 200
	cacheMaintInterval = 5 * time.Minute
	cacheInitialDelay  = 20 * time.Second
	cachePruneMetaKey  = "cache.pruned.day"
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
	cutoff := start.AddDate(0, 0, -(cacheRetentionDays - 1))
	removed, err := s.st.PurgeTopicCacheBefore(ctx, cutoff.Unix())
	if err != nil {
		s.log.Warn("cache purge failed", "err", err)
		return
	}
	s.metaSetInt(ctx, cachePruneMetaKey, start.Unix())
	if removed > 0 {
		s.log.Info("old topic cache purged", "topics", removed, "cutoff", cutoff.Format("2006-01-02"))
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
	for _, id := range ids {
		select {
		case <-ctx.Done():
			return
		default:
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
			s.log.Warn("prefetch topic failed", "id", id, "err", err)
			continue
		}
		fetched++
	}
	if fetched > 0 || failed > 0 {
		s.log.Info("today topic cache prefetched", "fetched", fetched, "failed", failed, "total", len(ids))
	}
}
