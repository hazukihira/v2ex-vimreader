package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/hazukihira/v2ex-vimreader/internal/model"
)

// UpsertNodes inserts or updates the node directory.
func (s *Store) UpsertNodes(ctx context.Context, nodes []model.Node) error {
	if len(nodes) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO nodes (name, title, title_alt, parent, topics_count, restricted, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			title = CASE WHEN excluded.title <> "" THEN excluded.title ELSE nodes.title END,
			title_alt = CASE WHEN excluded.title_alt <> "" THEN excluded.title_alt ELSE nodes.title_alt END,
			parent = CASE WHEN excluded.parent <> "" THEN excluded.parent ELSE nodes.parent END,
			topics_count = CASE WHEN excluded.topics_count > 0 THEN excluded.topics_count ELSE nodes.topics_count END,
			restricted = CASE WHEN excluded.restricted <> 0 THEN 1 ELSE nodes.restricted END,
			fetched_at = excluded.fetched_at
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	ts := nowUnix()
	for _, n := range nodes {
		if n.Name == "" {
			continue
		}
		if _, err := stmt.ExecContext(ctx, n.Name, n.Title, n.TitleAlt, n.Parent, n.TopicsCount, boolToInt(n.Restricted), ts); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListNodes returns the whole node directory ordered by title.
func (s *Store) ListNodes(ctx context.Context) ([]model.Node, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT name, title, title_alt, parent, topics_count, restricted FROM nodes ORDER BY title COLLATE NOCASE, name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Node, 0, 64)
	for rows.Next() {
		var n model.Node
		var restricted int
		if err := rows.Scan(&n.Name, &n.Title, &n.TitleAlt, &n.Parent, &n.TopicsCount, &restricted); err != nil {
			return nil, err
		}
		n.Restricted = restricted != 0
		out = append(out, n)
	}
	return out, rows.Err()
}

// GetNode loads one node. Returns ErrNotFound when missing.
func (s *Store) GetNode(ctx context.Context, name string) (model.Node, error) {
	var n model.Node
	var restricted int
	err := s.db.QueryRowContext(ctx, "SELECT name, title, title_alt, parent, topics_count, restricted FROM nodes WHERE name = ?", name).
		Scan(&n.Name, &n.Title, &n.TitleAlt, &n.Parent, &n.TopicsCount, &restricted)
	if errors.Is(err, sql.ErrNoRows) {
		return n, ErrNotFound
	}
	n.Restricted = restricted != 0
	return n, err
}

// SetNodeRestricted marks a node as login-restricted.
func (s *Store) SetNodeRestricted(ctx context.Context, name string, restricted bool) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO nodes(name, title, restricted, fetched_at) VALUES(?, "", ?, ?)
		ON CONFLICT(name) DO UPDATE SET restricted = excluded.restricted
	`, name, boolToInt(restricted), nowUnix())
	return err
}

// NodeNameByTitle finds the node name for a display title.
func (s *Store) NodeNameByTitle(ctx context.Context, title string) (string, error) {
	var name string
	err := s.db.QueryRowContext(ctx, "SELECT name FROM nodes WHERE title = ? ORDER BY topics_count DESC LIMIT 1", title).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return name, err
}

// NodeCount returns the number of nodes in the directory.
func (s *Store) NodeCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM nodes").Scan(&n)
	return n, err
}
