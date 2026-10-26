package service

import (
	"context"
	"strings"
	"time"

	"github.com/hazukihira/v2ex-vimreader/internal/model"
	"github.com/hazukihira/v2ex-vimreader/internal/store"
	"github.com/hazukihira/v2ex-vimreader/internal/v2ex"
)

const siteFeedKey = "site:index"

func (s *Service) nodeDirectoryLoop(ctx context.Context) {
	s.runNodeDirectory(ctx)
	ticker := time.NewTicker(nodeDirectoryTTL)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runNodeDirectory(ctx)
		}
	}
}

func (s *Service) runNodeDirectory(ctx context.Context) {
	if err := s.syncNodeDirectory(ctx); err != nil {
		s.SetError(err)
		s.log.Warn("node directory sync failed", "err", err)
	}
}

func (s *Service) siteFeedLoop(ctx context.Context) {
	if s.cfg.Sync.SiteFeed {
		s.runSiteFeed(ctx)
	}
	interval := time.Duration(s.cfg.Fetch.SiteFeedPollSeconds) * time.Second
	if interval < time.Minute {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if s.cfg.Sync.SiteFeed {
				s.runSiteFeed(ctx)
			}
		}
	}
}

func (s *Service) runSiteFeed(ctx context.Context) {
	if err := s.syncSiteFeed(ctx); err != nil {
		s.SetError(err)
		s.log.Warn("site feed sync failed", "err", err)
	}
}

func (s *Service) nodeFeedsLoop(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(8 * time.Second):
	}
	s.runNodeFeeds(ctx)
	interval := time.Duration(s.cfg.Fetch.NodeFeedPollSeconds) * time.Second
	if interval < 5*time.Minute {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runNodeFeeds(ctx)
		}
	}
}

func (s *Service) runNodeFeeds(ctx context.Context) {
	for _, node := range s.FollowedNodeNames() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := s.syncNodeFeed(ctx, node); err != nil {
			s.SetError(err)
			s.log.Warn("node feed sync failed", "node", node, "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(900 * time.Millisecond):
		}
	}
	s.markNodeFeedsDone()
}

func (s *Service) syncSiteFeed(ctx context.Context) error {
	state, exists, err := s.st.GetFeedState(ctx, siteFeedKey)
	if err != nil {
		return err
	}
	var options []v2ex.Option
	if exists {
		options = append(options, v2ex.WithETag(state.ETag), v2ex.WithLastModified(state.LastModified))
	}
	resp, err := s.vc.Get(ctx, v2ex.IndexFeedURL(), options...)
	if err != nil {
		return err
	}
	if resp.NotModified {
		_ = s.st.TouchFeedState(ctx, siteFeedKey)
		s.markSiteFeedDone()
		return nil
	}
	if err := resp.Err(); err != nil {
		return err
	}
	topics, err := v2ex.ParseAtomFeed(resp.Body)
	if err != nil {
		return err
	}
	for i := range topics {
		s.storeFeedTopic(ctx, &topics[i])
	}
	now := time.Now().Unix()
	if err := s.st.SaveFeedState(ctx, store.FeedState{
		Key:          siteFeedKey,
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
		LastChecked:  now,
		LastSuccess:  now,
	}); err != nil {
		s.log.Warn("save site feed state failed", "err", err)
	}
	s.markSiteFeedDone()
	s.log.Info("site feed synced", "topics", len(topics))
	return nil
}

func (s *Service) syncNodeFeed(ctx context.Context, node string) error {
	if s.isRestrictedNode(ctx, node) {
		s.log.Debug("skip restricted node feed", "node", node)
		return nil
	}
	key := "node:" + node
	state, exists, err := s.st.GetFeedState(ctx, key)
	if err != nil {
		return err
	}
	var options []v2ex.Option
	if exists {
		options = append(options, v2ex.WithETag(state.ETag), v2ex.WithLastModified(state.LastModified))
	}
	resp, err := s.vc.Get(ctx, v2ex.NodeFeedURL(node), options...)
	if err != nil {
		return err
	}
	if resp.NotModified {
		_ = s.st.TouchFeedState(ctx, key)
		return nil
	}
	if resp.StatusCode == 404 {
		s.log.Debug("node feed is not available", "node", node)
		return nil
	}
	if err := resp.Err(); err != nil {
		return err
	}
	topics, err := v2ex.ParseJSONFeed(resp.Body, node)
	if err != nil {
		return err
	}
	title := s.nodeTitle(ctx, node)
	for i := range topics {
		if topics[i].NodeTitle == "" {
			topics[i].NodeTitle = title
		}
		s.storeFeedTopic(ctx, &topics[i])
	}
	now := time.Now().Unix()
	if err := s.st.SaveFeedState(ctx, store.FeedState{
		Key:          key,
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
		LastChecked:  now,
		LastSuccess:  now,
	}); err != nil {
		s.log.Warn("save node feed state failed", "node", node, "err", err)
	}
	s.log.Info("node feed synced", "node", node, "topics", len(topics))
	return nil
}

func (s *Service) storeFeedTopic(ctx context.Context, topic *model.Topic) {
	if strings.TrimSpace(topic.ContentHTML) != "" {
		topic.ContentHTML = s.san.Sanitize(topic.ContentHTML)
		topic.ContentText = strings.TrimSpace(v2ex.HTMLText(topic.ContentHTML))
	}
	if topic.Node == "" && topic.NodeTitle != "" {
		topic.Node = s.nodeNameByTitle(ctx, topic.NodeTitle)
	}
	if topic.NodeTitle == "" && topic.Node != "" {
		topic.NodeTitle = s.nodeTitle(ctx, topic.Node)
	}
	if topic.Node != "" && s.isRestrictedNode(ctx, topic.Node) {
		s.log.Debug("skip restricted feed topic", "id", topic.ID, "node", topic.Node)
		return
	}
	if err := s.st.UpsertTopic(ctx, *topic); err != nil {
		s.log.Warn("upsert feed topic failed", "id", topic.ID, "err", err)
	}
}
