package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/hazukihira/v2ex-vimreader/internal/model"
)

// SaveListPage stores a parsed site list page snapshot.
func (s *Store) SaveListPage(ctx context.Context, key string, page, pages int, entries []model.ListEntry) error {
	if entries == nil {
		entries = []model.ListEntry{}
	}
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	if pages < 1 {
		pages = 1
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO list_pages (list_key, page, entries, pages, fetched_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(list_key, page) DO UPDATE SET
			entries = excluded.entries, pages = excluded.pages, fetched_at = excluded.fetched_at
	`, key, page, string(data), pages, nowUnix())
	return err
}

// GetListPage loads a list page snapshot. The bool reports whether it exists.
func (s *Store) GetListPage(ctx context.Context, key string, page int) (model.ListPage, bool, error) {
	var data string
	var pages int
	var fetchedAt int64
	err := s.db.QueryRowContext(ctx, "SELECT entries, pages, fetched_at FROM list_pages WHERE list_key = ? AND page = ?", key, page).
		Scan(&data, &pages, &fetchedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.ListPage{}, false, nil
	}
	if err != nil {
		return model.ListPage{}, false, err
	}
	var entries []model.ListEntry
	if err := json.Unmarshal([]byte(data), &entries); err != nil {
		return model.ListPage{}, false, err
	}
	return model.ListPage{Entries: entries, Pages: pages, FetchedAt: fetchedAt}, true, nil
}
