package service

import (
	"context"
	"strings"

	"github.com/hazukihira/v2ex-vimreader/internal/model"
)

// Search looks for the query in cached titles, bodies and replies.
func (s *Service) Search(ctx context.Context, query string) (*SearchResponse, error) {
	query = strings.TrimSpace(query)
	if len(query) < 1 {
		return &SearchResponse{Topics: []model.ListEntry{}}, nil
	}
	topics, err := s.st.SearchTopics(ctx, query, 50)
	if err != nil {
		return nil, err
	}
	blockedTopics, blockedUsers, err := s.blockedSets(ctx)
	if err != nil {
		return nil, err
	}
	entries := make([]model.ListEntry, 0, len(topics))
	for _, topic := range topics {
		if blockedTopics[topic.ID] || blockedUsers[topic.Author] {
			continue
		}
		entries = append(entries, TopicToListEntry(topic))
	}
	return &SearchResponse{Topics: entries}, nil
}
