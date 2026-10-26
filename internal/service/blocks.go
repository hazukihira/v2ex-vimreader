package service

import (
	"context"
	"strconv"
	"strings"
)

// Blocks returns the local block list.
func (s *Service) Blocks(ctx context.Context) (*BlocksResponse, error) {
	b, err := s.st.ListBlocks(ctx)
	if err != nil {
		return nil, err
	}
	resp := &BlocksResponse{Topics: b.TopicIDs, Users: b.Users}
	if resp.Topics == nil {
		resp.Topics = []int64{}
	}
	if resp.Users == nil {
		resp.Users = []string{}
	}
	return resp, nil
}

// AddBlock adds one entry to the block list.
func (s *Service) AddBlock(ctx context.Context, kind string, value string) error {
	kind = strings.TrimSpace(strings.ToLower(kind))
	value = strings.TrimSpace(value)
	if value == "" {
		return ErrBadRequest
	}
	switch kind {
	case "topic":
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil || id <= 0 {
			return ErrBadRequest
		}
		return s.st.AddBlock(ctx, kind, strconv.FormatInt(id, 10))
	case "user":
		return s.st.AddBlock(ctx, kind, value)
	default:
		return ErrBadRequest
	}
}

// RemoveBlock removes one entry from the block list.
func (s *Service) RemoveBlock(ctx context.Context, kind string, value string) error {
	kind = strings.TrimSpace(strings.ToLower(kind))
	value = strings.TrimSpace(value)
	if kind != "topic" && kind != "user" {
		return ErrBadRequest
	}
	if value == "" {
		return ErrBadRequest
	}
	return s.st.RemoveBlock(ctx, kind, value)
}

func (s *Service) blockedSets(ctx context.Context) (map[int64]bool, map[string]bool, error) {
	b, err := s.st.ListBlocks(ctx)
	if err != nil {
		return nil, nil, err
	}
	topics := make(map[int64]bool, len(b.TopicIDs))
	for _, id := range b.TopicIDs {
		topics[id] = true
	}
	users := make(map[string]bool, len(b.Users))
	for _, name := range b.Users {
		users[name] = true
	}
	return topics, users, nil
}
