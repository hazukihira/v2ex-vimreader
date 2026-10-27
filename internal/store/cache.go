package store

import (
	"context"
)

// TopicIDsForWarmup lists topics created in [from, until) that do not have a
// complete cached page yet. Content imported from feeds still has fetched_at
// zero, so those rows are included until a topic page is actually fetched.
func (s *Store) TopicIDsForWarmup(ctx context.Context, from, until int64, limit int) ([]int64, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id FROM topics
		WHERE ((created_at >= ? AND created_at < ?) OR (created_at = 0 AND feed_at >= ? AND feed_at < ?))
		  AND (content_html = "" OR fetched_at = 0)
		ORDER BY created_at ASC, id ASC
		LIMIT ?`, from, until, from, until, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]int64, 0, limit)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// PurgeTopicCacheBefore removes cached bodies and replies for topics older than
// cutoff. Topic metadata stays in the archive so day/new lists keep working;
// opening an old topic will fetch it again on demand.
func (s *Store) PurgeTopicCacheBefore(ctx context.Context, cutoff int64) (int64, error) {
	const effectiveTime = `CASE WHEN created_at > 0 THEN created_at WHEN feed_at > 0 THEN feed_at ELSE fetched_at END`
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `
		UPDATE topics
		SET content_html = "", content_text = "", fetched_at = 0
		WHERE `+effectiveTime+` < ?
		  AND (content_html <> "" OR content_text <> "" OR fetched_at <> 0)`, cutoff)
	if err != nil {
		return 0, err
	}
	removed, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM replies
		WHERE topic_id IN (SELECT id FROM topics WHERE `+effectiveTime+` < ?)`, cutoff); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM list_pages WHERE fetched_at < ?", cutoff); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return removed, nil
}

// PurgeArchiveBefore deletes old topic metadata and replies. This keeps the
// archive table bounded; day/new views only cover the retained window.
func (s *Store) PurgeArchiveBefore(ctx context.Context, cutoff int64) (int64, error) {
	const effectiveTime = `CASE WHEN created_at > 0 THEN created_at WHEN feed_at > 0 THEN feed_at ELSE fetched_at END`
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM summaries
		WHERE topic_id IN (SELECT id FROM topics WHERE `+effectiveTime+` < ?)`, cutoff); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM replies
		WHERE topic_id IN (SELECT id FROM topics WHERE `+effectiveTime+` < ?)`, cutoff); err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, "DELETE FROM topics WHERE "+effectiveTime+" < ?", cutoff)
	if err != nil {
		return 0, err
	}
	removed, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return removed, nil
}

// PurgeSummariesBefore drops AI summaries whose cache entry is older than cutoff.
func (s *Store) PurgeSummariesBefore(ctx context.Context, cutoff int64) (int64, error) {
	result, err := s.db.ExecContext(ctx, "DELETE FROM summaries WHERE created_at < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// PurgeNodesBefore drops node-directory entries not refreshed since cutoff.
func (s *Store) PurgeNodesBefore(ctx context.Context, cutoff int64) (int64, error) {
	result, err := s.db.ExecContext(ctx, "DELETE FROM nodes WHERE fetched_at < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// PurgeFeedStatesBefore drops conditional-request state for feeds that have not
// been checked since cutoff.
func (s *Store) PurgeFeedStatesBefore(ctx context.Context, cutoff int64) (int64, error) {
	result, err := s.db.ExecContext(ctx, "DELETE FROM feed_state WHERE last_checked < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
