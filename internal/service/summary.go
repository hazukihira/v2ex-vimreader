package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hazukihira/v2ex-vimreader/internal/ai"
	"github.com/hazukihira/v2ex-vimreader/internal/model"
	"github.com/hazukihira/v2ex-vimreader/internal/v2ex"
)

const summaryTTL = 24 * time.Hour

const defaultSummaryPrompt = "你是一个中文论坛阅读助手。请用简洁的中文要点总结这个 V2EX 帖子：" +
	"1) 楼主想表达什么；2) 回帖中的主要观点、分歧或补充信息；3) 简短结论。" +
	"只依据给出的内容，不要编造；控制在 200 字以内。"

// Summary returns the cached summary state of a topic.
func (s *Service) Summary(ctx context.Context, id int64) (*SummaryDTO, error) {
	if id <= 0 {
		return nil, ErrBadRequest
	}
	summary := s.summaryDTO(ctx, id)
	return &summary, nil
}

// GenerateSummary creates (or streams) an AI summary for a topic.
func (s *Service) GenerateSummary(ctx context.Context, id int64, refresh bool, onDelta func(string) error) (*SummaryDTO, error) {
	if id <= 0 {
		return nil, ErrBadRequest
	}
	if !s.ai.Enabled() {
		return nil, ai.ErrDisabled
	}
	if !refresh {
		if cached, ok, err := s.st.GetSummary(ctx, id); err == nil && ok && isFresh(cached.CreatedAt, summaryTTL) {
			if onDelta != nil {
				if err := onDelta(cached.Content); err != nil {
					return nil, err
				}
			}
			return &SummaryDTO{Status: "ready", Content: cached.Content, Model: cached.Model, CreatedAt: cached.CreatedAt}, nil
		}
	}
	topic, err := s.st.GetTopic(ctx, id)
	if err != nil {
		if _, fetchErr := s.Topic(ctx, id, 1, false); fetchErr != nil {
			return nil, fetchErr
		}
		topic, err = s.st.GetTopic(ctx, id)
		if err != nil {
			return nil, ErrNotFound
		}
	}
	if strings.TrimSpace(topic.ContentText) == "" || !isFresh(topic.FetchedAt, s.cfg.Fetch.TopicCacheTTL()) {
		if _, err := s.Topic(ctx, id, 1, false); err != nil {
			return nil, err
		}
		topic, err = s.st.GetTopic(ctx, id)
		if err != nil {
			return nil, err
		}
	}
	replies, err := s.st.RepliesForSummary(ctx, id, 100)
	if err != nil {
		return nil, err
	}
	system, user := s.buildSummaryPrompt(topic, replies)
	content, err := s.ai.Generate(ctx, system, user, onDelta)
	if err != nil {
		return nil, err
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("ai returned an empty summary")
	}
	if err := s.st.PutSummary(ctx, id, s.ai.Model(), content); err != nil {
		s.log.Warn("save summary failed", "id", id, "err", err)
	}
	return &SummaryDTO{Status: "ready", Content: content, Model: s.ai.Model(), CreatedAt: time.Now().Unix()}, nil
}

func (s *Service) buildSummaryPrompt(topic model.Topic, replies []model.Reply) (string, string) {
	system := strings.TrimSpace(s.cfg.AI.Prompt)
	if system == "" {
		system = defaultSummaryPrompt
	}
	var b strings.Builder
	fmt.Fprintf(&b, "帖子标题：%s\n", topic.Title)
	fmt.Fprintf(&b, "节点：%s\n作者：%s\n链接：%s\n\n", topic.NodeTitle, topic.Author, v2ex.TopicWebURL(topic.ID))
	b.WriteString("楼主正文：\n")
	b.WriteString(topic.ContentText)
	b.WriteString("\n\n回帖节选：\n")
	for _, reply := range replies {
		if reply.ContentText == "" {
			continue
		}
		if b.Len() >= s.cfg.AI.MaxInputChars {
			break
		}
		fmt.Fprintf(&b, "#%d %s：%s\n", reply.Floor, reply.Author, reply.ContentText)
	}
	user := truncateRunes(b.String(), s.cfg.AI.MaxInputChars)
	return system, user
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "\n...(已截断)"
}
