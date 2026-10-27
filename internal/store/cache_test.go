package store

import (
	"context"
	"testing"
	"time"

	"github.com/hazukihira/v2ex-vimreader/internal/model"
)

func TestPurgeTopicCacheBeforeKeepsArchiveMetadata(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	now := time.Now().Unix()
	old := now - 10*24*3600
	recent := now - 3600
	for _, topic := range []model.Topic{
		{ID: 1, Node: "tech", Title: "old", Author: "alice", CreatedAt: old, Replies: 1, ContentHTML: "<p>old</p>", ContentText: "old", FetchedAt: now},
		{ID: 2, Node: "tech", Title: "recent", Author: "bob", CreatedAt: recent, Replies: 1, ContentHTML: "<p>recent</p>", ContentText: "recent", FetchedAt: now},
	} {
		if err := st.UpsertTopic(ctx, topic); err != nil {
			t.Fatal(err)
		}
		if err := st.UpsertReplies(ctx, topic.ID, []model.Reply{{Floor: 1, Author: "bob", ContentHTML: "<p>reply</p>", ContentText: "reply", FetchedAt: now}}); err != nil {
			t.Fatal(err)
		}
	}
	cutoff := now - 3*24*3600
	removed, err := st.PurgeTopicCacheBefore(ctx, cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("want 1 purged topic, got %d", removed)
	}
	gotOld, err := st.GetTopic(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if gotOld.Title != "old" || gotOld.ContentHTML != "" || gotOld.ContentText != "" || gotOld.FetchedAt != 0 {
		t.Fatalf("old topic cache not purged correctly: %+v", gotOld)
	}
	gotRecent, err := st.GetTopic(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if gotRecent.ContentHTML != "<p>recent</p>" || gotRecent.FetchedAt != now {
		t.Fatalf("recent topic should stay cached: %+v", gotRecent)
	}
	if n, err := st.RepliesCount(ctx, 1); err != nil || n != 0 {
		t.Fatalf("old replies not removed: n=%d err=%v", n, err)
	}
	if n, err := st.RepliesCount(ctx, 2); err != nil || n != 1 {
		t.Fatalf("recent replies removed: n=%d err=%v", n, err)
	}
}

func TestTopicIDsForWarmupSkipsFetchedPages(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	now := time.Now().Unix()
	from, until := now-3600, now+3600
	for _, topic := range []model.Topic{
		{ID: 11, Node: "tech", CreatedAt: now, ContentHTML: "<p>feed</p>", FetchedAt: 0},
		{ID: 12, Node: "tech", CreatedAt: now, ContentHTML: "<p>done</p>", FetchedAt: now},
		{ID: 13, Node: "tech", CreatedAt: now, ContentHTML: "", FetchedAt: 0},
		{ID: 14, Node: "tech", CreatedAt: now - 2*3600, ContentHTML: "<p>old</p>", FetchedAt: 0},
	} {
		if err := st.UpsertTopic(ctx, topic); err != nil {
			t.Fatal(err)
		}
	}
	ids, err := st.TopicIDsForWarmup(ctx, from, until, 10)
	if err != nil {
		t.Fatal(err)
	}
	want := []int64{11, 13}
	if len(ids) != len(want) {
		t.Fatalf("want ids %v, got %v", want, ids)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("want ids %v, got %v", want, ids)
		}
	}
}

func TestPurgeOtherCacheLayers(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	now := time.Now().Unix()
	old := now - 40*24*3600
	if err := st.UpsertTopic(ctx, model.Topic{ID: 21, Node: "tech", CreatedAt: old, ContentHTML: "<p>old</p>", FetchedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertNodes(ctx, []model.Node{{Name: "oldnode", Title: "old"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.ExecContext(ctx, "UPDATE nodes SET fetched_at = ? WHERE name = 'oldnode'", old); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveFeedState(ctx, FeedState{Key: "oldfeed", LastChecked: old, LastSuccess: old}); err != nil {
		t.Fatal(err)
	}
	if n, err := st.PurgeArchiveBefore(ctx, now-30*24*3600); err != nil || n != 1 {
		t.Fatalf("archive purge n=%d err=%v", n, err)
	}
	if err := st.PutSummary(ctx, 21, "m", "summary"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.ExecContext(ctx, "UPDATE summaries SET created_at = ? WHERE topic_id = 21", old); err != nil {
		t.Fatal(err)
	}
	if n, err := st.PurgeNodesBefore(ctx, now-30*24*3600); err != nil || n != 1 {
		t.Fatalf("node purge n=%d err=%v", n, err)
	}
	if n, err := st.PurgeFeedStatesBefore(ctx, now-30*24*3600); err != nil || n != 1 {
		t.Fatalf("feed purge n=%d err=%v", n, err)
	}
	if n, err := st.PurgeSummariesBefore(ctx, now-24*3600); err != nil || n != 1 {
		t.Fatalf("summary purge n=%d err=%v", n, err)
	}
}
