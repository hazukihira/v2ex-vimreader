package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/hazukihira/v2ex-vimreader/internal/model"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestTopicUpsertKeepsKnownFields(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	created := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC).Unix()
	topic := model.Topic{
		ID: 100, Node: "tech", NodeTitle: "科技", Title: "hello", Author: "alice",
		CreatedAt: created, Replies: 3, ContentHTML: "<p>hi</p>", ContentText: "hi",
	}
	if err := st.UpsertTopic(ctx, topic); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertTopic(ctx, model.Topic{ID: 100, Replies: 5, LastTouchedAt: created + 60, FeedAt: created + 10}); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetTopic(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "hello" || got.ContentHTML != "<p>hi</p>" {
		t.Fatalf("known fields were erased: %+v", got)
	}
	if got.Replies != 5 || got.LastTouchedAt != created+60 {
		t.Fatalf("new fields not merged: %+v", got)
	}
	if got.CreatedAt != created {
		t.Fatalf("created_at changed: %d", got.CreatedAt)
	}
}

func TestListDayAndNew(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	loc := time.UTC
	start, end, err := DayRange("2026-10-25", loc)
	if err != nil {
		t.Fatal(err)
	}
	for i, ts := range []int64{start + 10, start + 3600, end + 100} {
		topic := model.Topic{ID: int64(i + 1), Node: "qna", Title: "topic", CreatedAt: ts, Replies: 0}
		if err := st.UpsertTopic(ctx, topic); err != nil {
			t.Fatal(err)
		}
	}
	topics, total, err := st.ListDay(ctx, start, end, 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(topics) != 2 {
		t.Fatalf("want 2 topics in day, got total=%d len=%d", total, len(topics))
	}
	if topics[0].ID != 2 {
		t.Fatalf("want newest first, got id %d", topics[0].ID)
	}
	all, allTotal, err := st.ListNew(ctx, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if allTotal != 3 || len(all) != 3 {
		t.Fatalf("want 3 archived topics, got total=%d len=%d", allTotal, len(all))
	}
	from, to, err := st.ArchiveRange(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if from != start+10 || to != end+100 {
		t.Fatalf("unexpected archive range: %d..%d", from, to)
	}
	coverageFrom, coverageTo, err := st.CoverageRange(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if coverageFrom != 0 || coverageTo != 0 {
		t.Fatalf("coverage must ignore topic-only rows: %d..%d", coverageFrom, coverageTo)
	}
	if err := st.UpsertTopic(ctx, model.Topic{ID: 10, Node: "qna", Title: "feed", CreatedAt: start + 50, FeedAt: start + 50}); err != nil {
		t.Fatal(err)
	}
	coverageFrom, coverageTo, err = st.CoverageRange(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if coverageFrom != start+50 || coverageTo != start+50 {
		t.Fatalf("unexpected coverage range: %d..%d", coverageFrom, coverageTo)
	}
}

func TestReplies(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	replies := []model.Reply{
		{Floor: 1, Author: "a", CreatedAt: 100, ContentHTML: "<p>one</p>"},
		{Floor: 2, Author: "b", CreatedAt: 200, ContentHTML: "<p>two</p>"},
		{Floor: 3, Author: "a", CreatedAt: 300, ContentHTML: "<p>three</p>"},
	}
	if err := st.UpsertReplies(ctx, 7, replies); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertReplies(ctx, 7, []model.Reply{{Floor: 2, Author: "b", ContentHTML: "<p>two edited</p>"}}); err != nil {
		t.Fatal(err)
	}
	page, err := st.RepliesPage(ctx, 7, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 2 || page[0].Floor != 1 || page[1].Floor != 2 {
		t.Fatalf("unexpected page: %+v", page)
	}
	if page[1].ContentHTML != "<p>two edited</p>" {
		t.Fatalf("reply was not updated: %q", page[1].ContentHTML)
	}
	floor, err := st.MaxReplyFloor(ctx, 7)
	if err != nil || floor != 3 {
		t.Fatalf("want max floor 3, got %d (%v)", floor, err)
	}
	count, err := st.RepliesCount(ctx, 7)
	if err != nil || count != 3 {
		t.Fatalf("want 3 replies, got %d (%v)", count, err)
	}
}

func TestListPagesAndBlocks(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	entries := []model.ListEntry{{ID: 1, Title: "one", Node: "tech", Replies: 3}, {ID: 2, Title: "two"}}
	if err := st.SaveListPage(ctx, "node:tech", 1, 30, entries); err != nil {
		t.Fatal(err)
	}
	page, ok, err := st.GetListPage(ctx, "node:tech", 1)
	if err != nil || !ok {
		t.Fatalf("get list page: ok=%v err=%v", ok, err)
	}
	if len(page.Entries) != 2 || page.Pages != 30 || page.Entries[0].Title != "one" {
		t.Fatalf("unexpected list page: %+v", page)
	}
	if err := st.AddBlock(ctx, "topic", "123"); err != nil {
		t.Fatal(err)
	}
	if err := st.AddBlock(ctx, "user", "spammer"); err != nil {
		t.Fatal(err)
	}
	if err := st.AddBlock(ctx, "user", "spammer"); err != nil {
		t.Fatal(err)
	}
	blocked, err := st.ListBlocks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocked.TopicIDs) != 1 || blocked.TopicIDs[0] != 123 || len(blocked.Users) != 1 {
		t.Fatalf("unexpected blocks: %+v", blocked)
	}
	if err := st.RemoveBlock(ctx, "topic", "123"); err != nil {
		t.Fatal(err)
	}
	blocked, _ = st.ListBlocks(ctx)
	if len(blocked.TopicIDs) != 0 {
		t.Fatalf("topic block not removed: %+v", blocked)
	}
}

func TestDeleteTopicsByNodes(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	for _, topic := range []model.Topic{
		{ID: 1, Node: "nsfw", Title: "restricted", CreatedAt: 100},
		{ID: 2, Node: "tech", Title: "public", CreatedAt: 100},
	} {
		if err := st.UpsertTopic(ctx, topic); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.UpsertReplies(ctx, 1, []model.Reply{{Floor: 1, ContentText: "secret"}}); err != nil {
		t.Fatal(err)
	}
	removed, err := st.DeleteTopicsByNodes(ctx, []string{"nsfw", "chamber"})
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("want 1 removed topic, got %d", removed)
	}
	if exists, err := st.TopicExists(ctx, 1); err != nil || exists {
		t.Fatalf("restricted topic still present: exists=%v err=%v", exists, err)
	}
	if exists, err := st.TopicExists(ctx, 2); err != nil || !exists {
		t.Fatalf("public topic was removed: exists=%v err=%v", exists, err)
	}
	if count, err := st.RepliesCount(ctx, 1); err != nil || count != 0 {
		t.Fatalf("restricted replies not removed: count=%d err=%v", count, err)
	}
}

func TestFeedStateSummaryAndNodes(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	if err := st.SaveFeedState(ctx, FeedState{Key: "node:tech", ETag: "abc", LastChecked: 10, LastSuccess: 10}); err != nil {
		t.Fatal(err)
	}
	state, ok, err := st.GetFeedState(ctx, "node:tech")
	if err != nil || !ok || state.ETag != "abc" {
		t.Fatalf("unexpected feed state: %+v ok=%v err=%v", state, ok, err)
	}
	if err := st.PutSummary(ctx, 9, "model-x", "summary text"); err != nil {
		t.Fatal(err)
	}
	sum, ok, err := st.GetSummary(ctx, 9)
	if err != nil || !ok || sum.Content != "summary text" {
		t.Fatalf("unexpected summary: %+v ok=%v err=%v", sum, ok, err)
	}
	if err := st.UpsertNodes(ctx, []model.Node{{Name: "tech", Title: "科技"}, {Name: "nsfw", Title: "NSFW"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.SetNodeRestricted(ctx, "nsfw", true); err != nil {
		t.Fatal(err)
	}
	nodes, err := st.ListNodes(ctx)
	if err != nil || len(nodes) != 2 {
		t.Fatalf("unexpected nodes: %+v err=%v", nodes, err)
	}
	if !nodes[0].Restricted && !nodes[1].Restricted {
		t.Fatalf("restricted flag not stored: %+v", nodes)
	}
	name, err := st.NodeNameByTitle(ctx, "科技")
	if err != nil || name != "tech" {
		t.Fatalf("want tech, got %q (%v)", name, err)
	}
}

func TestSearchTopics(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	if err := st.UpsertTopic(ctx, model.Topic{ID: 1, Title: "unrelated", ContentText: "body", CreatedAt: 100}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertTopic(ctx, model.Topic{ID: 2, Title: "Golang channels", ContentText: "goroutine", CreatedAt: 200}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertReplies(ctx, 1, []model.Reply{{Floor: 1, ContentText: "mentioning goroutine here"}}); err != nil {
		t.Fatal(err)
	}
	topics, err := st.SearchTopics(ctx, "goroutine", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(topics) != 2 {
		t.Fatalf("want 2 matches, got %d", len(topics))
	}
}
