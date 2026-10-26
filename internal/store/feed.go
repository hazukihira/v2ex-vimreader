package store

import (
	"context"
	"database/sql"
	"errors"
)

// FeedState is the conditional-request state of one feed.
type FeedState struct {
	Key          string
	ETag         string
	LastModified string
	LastChecked  int64
	LastSuccess  int64
}

// GetFeedState loads a feed state. The bool reports whether it exists.
func (s *Store) GetFeedState(ctx context.Context, key string) (FeedState, bool, error) {
	var fs FeedState
	err := s.db.QueryRowContext(ctx, "SELECT feed_key, etag, last_modified, last_checked, last_success FROM feed_state WHERE feed_key = ?", key).
		Scan(&fs.Key, &fs.ETag, &fs.LastModified, &fs.LastChecked, &fs.LastSuccess)
	if errors.Is(err, sql.ErrNoRows) {
		return FeedState{Key: key}, false, nil
	}
	return fs, err == nil, err
}

// SaveFeedState stores the feed state update.
func (s *Store) SaveFeedState(ctx context.Context, fs FeedState) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO feed_state(feed_key, etag, last_modified, last_checked, last_success)
		VALUES(?, ?, ?, ?, ?)
		ON CONFLICT(feed_key) DO UPDATE SET
			etag = excluded.etag, last_modified = excluded.last_modified,
			last_checked = excluded.last_checked, last_success = excluded.last_success
	`, fs.Key, fs.ETag, fs.LastModified, fs.LastChecked, fs.LastSuccess)
	return err
}

// TouchFeedState records a 304 check without changing the stored validators.
func (s *Store) TouchFeedState(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE feed_state SET last_checked = ? WHERE feed_key = ?", nowUnix(), key)
	return err
}
