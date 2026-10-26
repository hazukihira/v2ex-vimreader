package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/hazukihira/v2ex-vimreader/internal/model"
)

const followedNodesMeta = "followed_nodes"

var nodeNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)

// loadFollowedNodes restores the sidebar node list from the database, falling
// back to the configured default when the user has never edited it.
func (s *Service) loadFollowedNodes() {
	ctx := context.Background()
	names := normalizeFollowedNames(s.cfg.Sync.FollowedNodes)
	if raw, err := s.st.GetMeta(ctx, followedNodesMeta); err == nil && raw != "" {
		var stored []string
		if err := json.Unmarshal([]byte(raw), &stored); err == nil {
			names = normalizeFollowedNames(stored)
		}
	}
	s.followedMu.Lock()
	s.followed = names
	s.followedMu.Unlock()
}

// SetFollowedNodes validates, stores and returns the new sidebar node list.
func (s *Service) SetFollowedNodes(ctx context.Context, names []string) ([]string, []model.Node, error) {
	cleaned := normalizeFollowedNames(names)
	if len(cleaned) > 100 {
		return nil, nil, fmt.Errorf("%w: 最多选择 100 个节点", ErrBadRequest)
	}
	infos := make([]model.Node, 0, len(cleaned))
	for _, name := range cleaned {
		if !nodeNamePattern.MatchString(name) {
			return nil, nil, fmt.Errorf("%w: 节点名不合法：%s", ErrInvalidNode, name)
		}
		node, err := s.st.GetNode(ctx, name)
		if err != nil {
			return nil, nil, fmt.Errorf("%w：%s（先在节点目录中刷新）", ErrInvalidNode, name)
		}
		infos = append(infos, node)
	}
	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return nil, nil, err
	}
	if err := s.st.SetMeta(ctx, followedNodesMeta, string(encoded)); err != nil {
		return nil, nil, err
	}
	s.followedMu.Lock()
	s.followed = cleaned
	s.followedMu.Unlock()
	return cleaned, infos, nil
}

// FollowedNodeInfos returns node metadata for the current sidebar names.
func (s *Service) FollowedNodeInfos(ctx context.Context) ([]string, []model.Node, error) {
	names := s.FollowedNodeNames()
	infos := make([]model.Node, 0, len(names))
	for _, name := range names {
		node, err := s.st.GetNode(ctx, name)
		if err != nil {
			node = model.Node{Name: name, Title: name}
		}
		infos = append(infos, node)
	}
	return names, infos, nil
}

func normalizeFollowedNames(names []string) []string {
	if len(names) == 0 {
		return []string{}
	}
	seen := make(map[string]bool, len(names))
	out := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}
