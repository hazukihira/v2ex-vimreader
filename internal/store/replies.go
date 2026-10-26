package store

import (
	"context"

	"github.com/hazukihira/v2ex-vimreader/internal/model"
)

// UpsertReplies inserts or updates one page of replies in a single transaction.
func (s *Store) UpsertReplies(ctx context.Context, topicID int64, replies []model.Reply) error {
	if len(replies) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO replies (topic_id, floor, author, created_at, content_html, content_text, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(topic_id, floor) DO UPDATE SET
			author = CASE WHEN excluded.author <> "" THEN excluded.author ELSE replies.author END,
			created_at = CASE WHEN excluded.created_at > 0 THEN excluded.created_at ELSE replies.created_at END,
			content_html = CASE WHEN excluded.content_html <> "" THEN excluded.content_html ELSE replies.content_html END,
			content_text = CASE WHEN excluded.content_text <> "" THEN excluded.content_text ELSE replies.content_text END,
			fetched_at = excluded.fetched_at
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, r := range replies {
		if r.Floor <= 0 {
			continue
		}
		if _, err := stmt.ExecContext(ctx, topicID, r.Floor, r.Author, r.CreatedAt, r.ContentHTML, r.ContentText, r.FetchedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// RepliesPage returns one page of replies ordered by floor.
func (s *Store) RepliesPage(ctx context.Context, topicID int64, page, pageSize int) ([]model.Reply, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT floor, author, created_at, content_html, content_text FROM replies WHERE topic_id = ? ORDER BY floor LIMIT ? OFFSET ?",
		topicID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Reply, 0, pageSize)
	for rows.Next() {
		var r model.Reply
		r.TopicID = topicID
		if err := rows.Scan(&r.Floor, &r.Author, &r.CreatedAt, &r.ContentHTML, &r.ContentText); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// RepliesCount returns the number of cached floors for a topic.
func (s *Store) RepliesCount(ctx context.Context, topicID int64) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM replies WHERE topic_id = ?", topicID).Scan(&n)
	return n, err
}

// MaxReplyFloor returns the largest cached floor of a topic (0 when none).
func (s *Store) MaxReplyFloor(ctx context.Context, topicID int64) (int, error) {
	var floor int
	err := s.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(floor), 0) FROM replies WHERE topic_id = ?", topicID).Scan(&floor)
	return floor, err
}

// RepliesForSummary returns up to limit replies for the AI prompt, ordered by floor.
func (s *Store) RepliesForSummary(ctx context.Context, topicID int64, limit int) ([]model.Reply, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.RepliesPage(ctx, topicID, 1, limit)
}
