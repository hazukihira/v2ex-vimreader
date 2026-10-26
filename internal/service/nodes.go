package service

import (
	"context"
	"time"

	"bytes"
	"github.com/PuerkitoBio/goquery"

	"github.com/hazukihira/v2ex-vimreader/internal/v2ex"
)

// Nodes returns the node directory, refreshing it when stale.
func (s *Service) Nodes(ctx context.Context, refresh bool) (*NodesResponse, error) {
	count, err := s.st.NodeCount(ctx)
	if err != nil {
		return nil, err
	}
	fetchedAt := s.metaInt(ctx, "nodes_fetched_at")
	if refresh || count == 0 || !isFresh(fetchedAt, nodeDirectoryTTL) {
		if err := s.syncNodeDirectory(ctx); err != nil {
			s.SetError(err)
			if count == 0 {
				return nil, err
			}
			s.log.Warn("node directory refresh failed", "err", err)
		}
		fetchedAt = s.metaInt(ctx, "nodes_fetched_at")
	}
	nodes, err := s.st.ListNodes(ctx)
	if err != nil {
		return nil, err
	}
	return &NodesResponse{Nodes: nodes, FetchedAt: fetchedAt}, nil
}

// syncNodeDirectory fetches /planes and stores the node list.
func (s *Service) syncNodeDirectory(ctx context.Context) error {
	_, err := s.flights.Do("nodes", func() (any, error) {
		resp, err := s.vc.Get(ctx, v2ex.PlanesURL())
		if err != nil {
			return nil, err
		}
		if err := resp.Err(); err != nil {
			return nil, err
		}
		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(resp.Body))
		if err != nil {
			return nil, err
		}
		nodes := v2ex.ParsePlanes(doc)
		if len(nodes) == 0 {
			return nil, ErrUnavailable
		}
		if err := s.st.UpsertNodes(ctx, nodes); err != nil {
			return nil, err
		}
		for name := range s.restricted {
			if err := s.st.SetNodeRestricted(ctx, name, true); err != nil {
				s.log.Warn("mark restricted node failed", "node", name, "err", err)
			}
		}
		if !s.loggedIn() {
			if purged, err := s.st.DeleteTopicsByNodes(ctx, s.RestrictedNodeNames()); err != nil {
				s.log.Warn("purge restricted topics failed", "err", err)
			} else if purged > 0 {
				s.log.Info("purged cached topics from restricted nodes", "topics", purged)
			}
		}
		s.metaSetInt(ctx, "nodes_fetched_at", time.Now().Unix())
		s.log.Info("node directory refreshed", "nodes", len(nodes))
		return nil, nil
	})
	return err
}

func (s *Service) nodeTitle(ctx context.Context, name string) string {
	if name == "" {
		return ""
	}
	if n, err := s.st.GetNode(ctx, name); err == nil && n.Title != "" {
		return n.Title
	}
	return name
}

func (s *Service) nodeNameByTitle(ctx context.Context, title string) string {
	if title == "" {
		return ""
	}
	if name, err := s.st.NodeNameByTitle(ctx, title); err == nil {
		return name
	}
	return ""
}

// FollowedNodeNames returns the sidebar node names (database override or config default).
func (s *Service) FollowedNodeNames() []string {
	s.followedMu.RLock()
	defer s.followedMu.RUnlock()
	out := make([]string, len(s.followed))
	copy(out, s.followed)
	return out
}
