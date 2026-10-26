package service

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/hazukihira/v2ex-vimreader/internal/config"
	"github.com/hazukihira/v2ex-vimreader/internal/model"
	"github.com/hazukihira/v2ex-vimreader/internal/store"
)

func newTestService(t *testing.T) (*Service, *store.Store) {
	t.Helper()
	cfg := config.Default()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(&cfg, st, logger), st
}

func TestRestrictedNodesAreSkipped(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService(t)
	for _, name := range []string{"nsfw", "sandbox", "chamber", "flamewar", "flood", "monthly", "survey", "cheap"} {
		if !svc.isRestrictedNode(ctx, name) {
			t.Fatalf("node %s should be restricted", name)
		}
	}
	if svc.isRestrictedNode(ctx, "tech") {
		t.Fatalf("tech should not be restricted")
	}
	if err := st.SetNodeRestricted(ctx, "somenode", true); err != nil {
		t.Fatal(err)
	}
	if !svc.isRestrictedNode(ctx, "somenode") {
		t.Fatalf("runtime restricted flag was ignored")
	}

	svc.storeFeedTopic(ctx, &model.Topic{ID: 1, Node: "nsfw", Title: "should be skipped", CreatedAt: 100})
	if exists, err := st.TopicExists(ctx, 1); err != nil || exists {
		t.Fatalf("restricted topic was ingested: exists=%v err=%v", exists, err)
	}
	svc.storeFeedTopic(ctx, &model.Topic{ID: 2, Node: "tech", Title: "ok", CreatedAt: 100})
	if exists, err := st.TopicExists(ctx, 2); err != nil || !exists {
		t.Fatalf("normal topic was not ingested: exists=%v err=%v", exists, err)
	}
}
