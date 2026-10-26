package service

import (
	"context"
	"errors"
	"testing"

	"github.com/hazukihira/v2ex-vimreader/internal/config"
	"github.com/hazukihira/v2ex-vimreader/internal/model"
)

func TestSetFollowedNodesPersistsAndLoads(t *testing.T) {
	ctx := context.Background()
	cfg := config.Default()
	cfg.Sync.FollowedNodes = []string{"tech"}
	st := mustStore(t)
	if err := st.UpsertNodes(ctx, []model.Node{{Name: "tech", Title: "科技"}, {Name: "qna", Title: "问与答"}, {Name: "create", Title: "分享创造"}}); err != nil {
		t.Fatal(err)
	}
	svc := New(&cfg, st, discardLogger())
	if got := svc.FollowedNodeNames(); len(got) != 1 || got[0] != "tech" {
		t.Fatalf("config default not loaded: %v", got)
	}
	names, infos, err := svc.SetFollowedNodes(ctx, []string{"qna", "tech", "qna"})
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "qna" || names[1] != "tech" {
		t.Fatalf("unexpected normalized names: %v", names)
	}
	if len(infos) != 2 || infos[0].Title != "问与答" {
		t.Fatalf("unexpected infos: %+v", infos)
	}
	// A new Service must load the database override instead of the config.
	svc2 := New(&cfg, st, discardLogger())
	if got := svc2.FollowedNodeNames(); len(got) != 2 {
		t.Fatalf("persisted list not loaded: %v", got)
	}
	// Empty list is a valid choice and must override the config default.
	if _, _, err := svc2.SetFollowedNodes(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if got := svc2.FollowedNodeNames(); len(got) != 0 {
		t.Fatalf("empty list not persisted: %v", got)
	}
	svc3 := New(&cfg, st, discardLogger())
	if got := svc3.FollowedNodeNames(); len(got) != 0 {
		t.Fatalf("empty override lost: %v", got)
	}
}

func TestSetFollowedNodesRejectsUnknown(t *testing.T) {
	cfg := config.Default()
	st := mustStore(t)
	svc := New(&cfg, st, discardLogger())
	_, _, err := svc.SetFollowedNodes(context.Background(), []string{"does-not-exist"})
	if !errors.Is(err, ErrInvalidNode) {
		t.Fatalf("want ErrInvalidNode, got %v", err)
	}
}

func TestFollowedNodeNamesEmptyIsSlice(t *testing.T) {
	cfg := config.Default()
	st := mustStore(t)
	svc := New(&cfg, st, discardLogger())
	if _, _, err := svc.SetFollowedNodes(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if svc.FollowedNodeNames() == nil {
		t.Fatalf("empty followed list must be an empty slice, not nil")
	}
}
