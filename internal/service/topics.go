package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/hazukihira/v2ex-vimreader/internal/model"
	"github.com/hazukihira/v2ex-vimreader/internal/store"
	"github.com/hazukihira/v2ex-vimreader/internal/v2ex"
)

// Topic loads a topic and one page of replies, fetching from V2EX when needed.
func (s *Service) Topic(ctx context.Context, id int64, page int, refresh bool) (*TopicResponse, error) {
	if id <= 0 {
		return nil, ErrBadRequest
	}
	if page < 1 {
		page = 1
	}
	key := fmt.Sprintf("topic:%d:%d:%v", id, page, refresh)
	value, err := s.flights.Do(key, func() (any, error) {
		return s.loadTopic(ctx, id, page, refresh)
	})
	if err != nil {
		return nil, err
	}
	return value.(*TopicResponse), nil
}

func (s *Service) topicCacheUsable(topic model.Topic) bool {
	if topic.ContentHTML == "" || topic.FetchedAt == 0 {
		return false
	}
	if topic.CreatedAt > 0 {
		local := time.Unix(topic.CreatedAt, 0).In(s.loc)
		now := time.Now().In(s.loc)
		if local.Year() == now.Year() && local.YearDay() == now.YearDay() {
			return true
		}
	}
	return isFresh(topic.FetchedAt, s.cfg.Fetch.TopicCacheTTL())
}

func (s *Service) loadTopic(ctx context.Context, id int64, page int, refresh bool) (*TopicResponse, error) {
	topic, err := s.st.GetTopic(ctx, id)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	pages := s.topicPages(ctx, topic)
	needFetch := refresh || !s.topicCacheUsable(topic)
	if needFetch {
		parsed, err := s.fetchTopicPage(ctx, id, 1)
		if err != nil {
			return nil, err
		}
		topic = parsed.Topic
		pages = parsed.Pages
	}
	if page > pages {
		page = pages
	}
	if page < 1 {
		page = 1
	}
	replies, err := s.st.RepliesPage(ctx, id, page, replyPageSize)
	if err != nil {
		return nil, err
	}
	if len(replies) == 0 && topic.Replies != 0 {
		if _, err := s.fetchTopicPage(ctx, id, page); err != nil {
			return nil, err
		}
		replies, err = s.st.RepliesPage(ctx, id, page, replyPageSize)
		if err != nil {
			return nil, err
		}
	}
	blockedTopics, blockedUsers, err := s.blockedSets(ctx)
	if err != nil {
		return nil, err
	}
	_ = blockedTopics
	publicReplies := make([]ReplyDTO, 0, len(replies))
	for _, reply := range replies {
		if blockedUsers[reply.Author] {
			continue
		}
		publicReplies = append(publicReplies, ReplyDTO{
			Floor:       reply.Floor,
			Author:      reply.Author,
			CreatedAt:   reply.CreatedAt,
			ContentHTML: reply.ContentHTML,
			IsOp:        reply.Author == topic.Author,
		})
	}
	if publicReplies == nil {
		publicReplies = []ReplyDTO{}
	}
	return &TopicResponse{
		Topic: TopicDTO{
			ID:          topic.ID,
			Node:        topic.Node,
			NodeTitle:   topic.NodeTitle,
			Title:       topic.Title,
			Author:      topic.Author,
			CreatedAt:   topic.CreatedAt,
			Replies:     topic.Replies,
			Views:       topic.Views,
			ContentHTML: topic.ContentHTML,
			FetchedAt:   topic.FetchedAt,
		},
		Page:     page,
		Pages:    pages,
		PageSize: replyPageSize,
		Replies:  publicReplies,
		Summary:  s.summaryDTO(ctx, id),
	}, nil
}

// Replies returns one page of replies without reloading the topic header.
func (s *Service) Replies(ctx context.Context, id int64, page int, refresh bool) (*RepliesResponse, error) {
	if id <= 0 {
		return nil, ErrBadRequest
	}
	if page < 1 {
		page = 1
	}
	key := fmt.Sprintf("replies:%d:%d:%v", id, page, refresh)
	value, err := s.flights.Do(key, func() (any, error) {
		topic, err := s.st.GetTopic(ctx, id)
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			return nil, err
		}
		if topic.Replies < 0 || topic.FetchedAt == 0 {
			parsed, err := s.fetchTopicPage(ctx, id, 1)
			if err != nil {
				return nil, err
			}
			topic = parsed.Topic
		}
		pages := s.topicPages(ctx, topic)
		if page > pages {
			page = pages
		}
		replies, err := s.st.RepliesPage(ctx, id, page, replyPageSize)
		if err != nil {
			return nil, err
		}
		if len(replies) == 0 || refresh {
			if _, err := s.fetchTopicPage(ctx, id, page); err != nil {
				return nil, err
			}
			replies, err = s.st.RepliesPage(ctx, id, page, replyPageSize)
			if err != nil {
				return nil, err
			}
		}
		_, blockedUsers, err := s.blockedSets(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]ReplyDTO, 0, len(replies))
		for _, reply := range replies {
			if blockedUsers[reply.Author] {
				continue
			}
			out = append(out, ReplyDTO{
				Floor:       reply.Floor,
				Author:      reply.Author,
				CreatedAt:   reply.CreatedAt,
				ContentHTML: reply.ContentHTML,
				IsOp:        reply.Author == topic.Author,
			})
		}
		if out == nil {
			out = []ReplyDTO{}
		}
		return &RepliesResponse{Page: page, Pages: pages, PageSize: replyPageSize, Replies: out}, nil
	})
	if err != nil {
		return nil, err
	}
	return value.(*RepliesResponse), nil
}

const replyPageSize = 100

func (s *Service) topicPages(ctx context.Context, topic model.Topic) int {
	if topic.Replies >= 0 {
		pages := (topic.Replies + replyPageSize - 1) / replyPageSize
		if pages < 1 {
			pages = 1
		}
		return pages
	}
	floor, err := s.st.MaxReplyFloor(ctx, topic.ID)
	if err != nil || floor <= 0 {
		return 1
	}
	return (floor + replyPageSize - 1) / replyPageSize
}

func (s *Service) fetchTopicPage(ctx context.Context, id int64, page int) (*model.TopicPage, error) {
	resp, err := s.vc.Get(ctx, v2ex.TopicURL(id, page))
	if err != nil {
		s.SetError(err)
		return nil, err
	}
	if resp.IsLoginWall() {
		return nil, ErrLoginRequired
	}
	if err := resp.Err(); err != nil {
		var httpErr *v2ex.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == 404 {
			return nil, ErrNotFound
		}
		s.SetError(err)
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(resp.Body))
	if err != nil {
		return nil, err
	}
	parsed := v2ex.ParseTopicPage(doc, page)
	if parsed.Topic.ID == 0 {
		parsed.Topic.ID = id
	}
	if parsed.Topic.Node == "" && parsed.Topic.NodeTitle != "" {
		parsed.Topic.Node = s.nodeNameByTitle(ctx, parsed.Topic.NodeTitle)
	}
	if parsed.Topic.NodeTitle == "" && parsed.Topic.Node != "" {
		parsed.Topic.NodeTitle = s.nodeTitle(ctx, parsed.Topic.Node)
	}
	now := time.Now().Unix()
	parsed.Topic.FetchedAt = now
	parsed.Topic.ContentHTML = s.san.Sanitize(parsed.Topic.ContentHTML)
	parsed.Topic.ContentText = strings.TrimSpace(v2ex.HTMLText(parsed.Topic.ContentHTML))
	for i := range parsed.Replies {
		parsed.Replies[i].TopicID = id
		parsed.Replies[i].FetchedAt = now
		parsed.Replies[i].ContentHTML = s.san.Sanitize(parsed.Replies[i].ContentHTML)
		parsed.Replies[i].ContentText = strings.TrimSpace(v2ex.HTMLText(parsed.Replies[i].ContentHTML))
	}
	if err := s.st.UpsertTopic(ctx, parsed.Topic); err != nil {
		return nil, err
	}
	if err := s.st.UpsertReplies(ctx, id, parsed.Replies); err != nil {
		return nil, err
	}
	return parsed, nil
}

// RefreshTopic forces a refetch of one topic page.
func (s *Service) RefreshTopic(ctx context.Context, id int64, page int) (*TopicResponse, error) {
	return s.Topic(ctx, id, page, true)
}

func (s *Service) summaryDTO(ctx context.Context, id int64) SummaryDTO {
	sm, ok, err := s.st.GetSummary(ctx, id)
	if err != nil || !ok {
		return SummaryDTO{Status: "none"}
	}
	return SummaryDTO{Status: "ready", Content: sm.Content, Model: sm.Model, CreatedAt: sm.CreatedAt}
}
