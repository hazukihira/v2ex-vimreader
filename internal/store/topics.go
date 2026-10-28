package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hazukihira/v2ex-vimreader/internal/model"
)

const topicColumns = "id, node, node_title, title, author, created_at, last_touched_at, replies, views, content_html, content_text, fetched_at, feed_at"

func scanTopic(scanner interface{ Scan(...any) error }) (model.Topic, error) {
	var t model.Topic
	err := scanner.Scan(&t.ID, &t.Node, &t.NodeTitle, &t.Title, &t.Author,
		&t.CreatedAt, &t.LastTouchedAt, &t.Replies, &t.Views, &t.ContentHTML, &t.ContentText,
		&t.FetchedAt, &t.FeedAt)
	return t, err
}

// UpsertTopic inserts or merges a topic. Empty/new values never erase known data.
func (s *Store) UpsertTopic(ctx context.Context, t model.Topic) error {
	replies := t.Replies
	if replies < -1 {
		replies = -1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO topics (id, node, node_title, title, author, created_at, last_touched_at, replies, views, content_html, content_text, fetched_at, feed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			node = CASE WHEN excluded.node <> "" THEN excluded.node ELSE topics.node END,
			node_title = CASE WHEN excluded.node_title <> "" THEN excluded.node_title ELSE topics.node_title END,
			title = CASE WHEN excluded.title <> "" THEN excluded.title ELSE topics.title END,
			author = CASE WHEN excluded.author <> "" THEN excluded.author ELSE topics.author END,
			created_at = CASE WHEN excluded.created_at > 0 THEN excluded.created_at ELSE topics.created_at END,
			last_touched_at = CASE WHEN excluded.last_touched_at > 0 THEN excluded.last_touched_at ELSE topics.last_touched_at END,
			replies = CASE WHEN excluded.replies >= 0 THEN excluded.replies ELSE topics.replies END,
			views = CASE WHEN excluded.views > 0 THEN excluded.views ELSE topics.views END,
			content_html = CASE WHEN excluded.content_html <> "" THEN excluded.content_html ELSE topics.content_html END,
			content_text = CASE WHEN excluded.content_text <> "" THEN excluded.content_text ELSE topics.content_text END,
			fetched_at = CASE WHEN excluded.fetched_at > 0 THEN excluded.fetched_at ELSE topics.fetched_at END,
			feed_at = CASE WHEN excluded.feed_at > 0 THEN excluded.feed_at ELSE topics.feed_at END
	`, t.ID, t.Node, t.NodeTitle, t.Title, t.Author, t.CreatedAt, t.LastTouchedAt,
		replies, t.Views, t.ContentHTML, t.ContentText, t.FetchedAt, t.FeedAt)
	return err
}

// GetTopic loads a topic by id.
func (s *Store) GetTopic(ctx context.Context, id int64) (model.Topic, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+topicColumns+" FROM topics WHERE id = ?", id)
	t, err := scanTopic(row)
	if errors.Is(err, sql.ErrNoRows) {
		return t, ErrNotFound
	}
	return t, err
}

// TopicExists reports whether a topic row exists.
func (s *Store) TopicExists(ctx context.Context, id int64) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM topics WHERE id = ?", id).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// ArchiveRange returns the first and last creation time covered by the archive.
func (s *Store) ArchiveRange(ctx context.Context) (int64, int64, error) {
	var from, to sql.NullInt64
	err := s.db.QueryRowContext(ctx, "SELECT MIN(created_at), MAX(created_at) FROM topics WHERE created_at > 0").Scan(&from, &to)
	if err != nil {
		return 0, 0, err
	}
	return from.Int64, to.Int64, nil
}

// CoverageRange returns the creation-time range discovered through feeds. It
// is the honest "archive coverage" number: topics merely opened from a list
// page do not extend it.
func (s *Store) CoverageRange(ctx context.Context) (int64, int64, error) {
	var from, to sql.NullInt64
	err := s.db.QueryRowContext(ctx, "SELECT MIN(created_at), MAX(created_at) FROM topics WHERE created_at > 0 AND feed_at > 0").Scan(&from, &to)
	if err != nil {
		return 0, 0, err
	}
	return from.Int64, to.Int64, nil
}

// ListNode returns archived topics for a node ordered by last activity.
func (s *Store) ListNode(ctx context.Context, node string, page, pageSize int) ([]model.Topic, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM topics WHERE node = ? AND created_at > 0", node).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, "SELECT "+topicColumns+" FROM topics WHERE node = ? AND created_at > 0 ORDER BY last_touched_at DESC, id DESC LIMIT ? OFFSET ?",
		node, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]model.Topic, 0, pageSize)
	for rows.Next() {
		t, err := scanTopic(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}

// CountActiveBefore counts topics whose latest activity is older than before.
func (s *Store) CountActiveBefore(ctx context.Context, before int64) (int, error) {
	var total int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM topics WHERE created_at > 0 AND COALESCE(NULLIF(last_touched_at, 0), created_at) < ?", before).Scan(&total)
	return total, err
}

// ListActiveBefore returns archived topics older than before in activity order.
func (s *Store) ListActiveBefore(ctx context.Context, before int64, page, pageSize int) ([]model.Topic, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM topics WHERE created_at > 0 AND COALESCE(NULLIF(last_touched_at, 0), created_at) < ?", before).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, "SELECT "+topicColumns+" FROM topics WHERE created_at > 0 AND COALESCE(NULLIF(last_touched_at, 0), created_at) < ? ORDER BY COALESCE(NULLIF(last_touched_at, 0), created_at) DESC, id DESC LIMIT ? OFFSET ?", before, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]model.Topic, 0, pageSize)
	for rows.Next() {
		t, err := scanTopic(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}

// ListNew returns topics ordered by creation time (newest first), paginated.
func (s *Store) ListNew(ctx context.Context, page, pageSize int) ([]model.Topic, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM topics WHERE created_at > 0").Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, "SELECT "+topicColumns+" FROM topics WHERE created_at > 0 ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?",
		pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]model.Topic, 0, pageSize)
	for rows.Next() {
		t, err := scanTopic(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}

// ListDay returns topics created inside [start, end) ordered newest first.
func (s *Store) ListDay(ctx context.Context, start, end int64, page, pageSize int) ([]model.Topic, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM topics WHERE created_at >= ? AND created_at < ?", start, end).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, "SELECT "+topicColumns+" FROM topics WHERE created_at >= ? AND created_at < ? ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?",
		start, end, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]model.Topic, 0, pageSize)
	for rows.Next() {
		t, err := scanTopic(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}

// DayRange returns the local-day boundaries for date (YYYY-MM-DD) in loc.
func DayRange(date string, loc *time.Location) (int64, int64, error) {
	d, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid date %q, want YYYY-MM-DD", date)
	}
	return d.Unix(), d.Add(24 * time.Hour).Unix(), nil
}

// DeleteTopicsByNodes removes cached topics and their replies for the given
// node names. It is used to purge anything that slipped in from login-only
// nodes before the restricted-node filter was in place.
func (s *Store) DeleteTopicsByNodes(ctx context.Context, nodes []string) (int64, error) {
	if len(nodes) == 0 {
		return 0, nil
	}
	placeholders := make([]string, 0, len(nodes))
	args := make([]any, 0, len(nodes))
	for _, node := range nodes {
		node = strings.TrimSpace(node)
		if node == "" {
			continue
		}
		placeholders = append(placeholders, "?")
		args = append(args, node)
	}
	if len(args) == 0 {
		return 0, nil
	}
	in := strings.Join(placeholders, ", ")
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM replies WHERE topic_id IN (SELECT id FROM topics WHERE node IN ("+in+"))", args...); err != nil {
		return 0, err
	}
	res, err := tx.ExecContext(ctx, "DELETE FROM topics WHERE node IN ("+in+")", args...)
	if err != nil {
		return 0, err
	}
	affected, _ := res.RowsAffected()
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return affected, nil
}

// SearchTopics does a simple LIKE search over titles, bodies and replies.
func (s *Store) SearchTopics(ctx context.Context, q string, limit int) ([]model.Topic, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	like := "%" + q + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT t.id, t.node, t.node_title, t.title, t.author, t.created_at, t.last_touched_at, t.replies, t.views, "", "", t.fetched_at, t.feed_at
		FROM topics t LEFT JOIN replies r ON r.topic_id = t.id
		WHERE t.title LIKE ? OR t.content_text LIKE ? OR r.content_text LIKE ?
		ORDER BY t.created_at DESC LIMIT ?
	`, like, like, like, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Topic, 0, limit)
	for rows.Next() {
		t, err := scanTopic(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
