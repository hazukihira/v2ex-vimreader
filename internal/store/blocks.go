package store

import (
	"context"
	"fmt"

	"github.com/hazukihira/v2ex-vimreader/internal/model"
)

// Blocked is the local block list.
type Blocked struct {
	TopicIDs []int64
	Users    []string
}

// AddBlock adds a user or topic to the block list. kind is "user" or "topic".
func (s *Store) AddBlock(ctx context.Context, kind, value string) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO blocks(kind, value, created_at) VALUES(?, ?, ?) ON CONFLICT(kind, value) DO NOTHING",
		kind, value, nowUnix())
	return err
}

// RemoveBlock removes an entry from the block list.
func (s *Store) RemoveBlock(ctx context.Context, kind, value string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM blocks WHERE kind = ? AND value = ?", kind, value)
	return err
}

// ListBlocks returns the current block list.
func (s *Store) ListBlocks(ctx context.Context) (Blocked, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT kind, value FROM blocks ORDER BY created_at")
	if err != nil {
		return Blocked{}, err
	}
	defer rows.Close()
	var b Blocked
	for rows.Next() {
		var kind, value string
		if err := rows.Scan(&kind, &value); err != nil {
			return Blocked{}, err
		}
		switch kind {
		case "topic":
			var id int64
			if _, err := fmt.Sscanf(value, "%d", &id); err == nil {
				b.TopicIDs = append(b.TopicIDs, id)
			}
		case "user":
			b.Users = append(b.Users, value)
		}
	}
	return b, rows.Err()
}

// Summary is a cached AI summary.
type Summary struct {
	TopicID   int64
	Model     string
	Content   string
	CreatedAt int64
}

// GetSummary loads a cached summary.
func (s *Store) GetSummary(ctx context.Context, topicID int64) (Summary, bool, error) {
	var sm Summary
	err := s.db.QueryRowContext(ctx, "SELECT topic_id, model, content, created_at FROM summaries WHERE topic_id = ?", topicID).
		Scan(&sm.TopicID, &sm.Model, &sm.Content, &sm.CreatedAt)
	if err != nil {
		return Summary{}, false, err
	}
	return sm, true, nil
}

// PutSummary stores a summary.
func (s *Store) PutSummary(ctx context.Context, topicID int64, model, content string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO summaries(topic_id, model, content, created_at) VALUES(?, ?, ?, ?)
		ON CONFLICT(topic_id) DO UPDATE SET model = excluded.model, content = excluded.content, created_at = excluded.created_at
	`, topicID, model, content, nowUnix())
	return err
}

// Ensure model import is used even if the file evolves.
var _ = model.Node{}
