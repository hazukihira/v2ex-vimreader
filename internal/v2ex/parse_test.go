package v2ex

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func loadDoc(t *testing.T, name string) *goquery.Document {
	t.Helper()
	f, err := os.Open("testdata/" + name)
	if err != nil {
		t.Fatalf("open fixture %s: %v", name, err)
	}
	defer f.Close()
	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	return doc
}

func TestParseNodeList(t *testing.T) {
	page := ParseList(loadDoc(t, "go_tech.html"), KindNode)
	if len(page.Entries) != 20 {
		t.Fatalf("want 20 entries, got %d", len(page.Entries))
	}
	if page.Pages != 30 {
		t.Fatalf("want 30 pages, got %d", page.Pages)
	}
	first := page.Entries[0]
	if first.ID != 1251278 {
		t.Fatalf("want topic 1251278, got %d", first.ID)
	}
	if first.Node != "" {
		t.Fatalf("node page rows have no node link, got %q", first.Node)
	}
	if first.Author != "grandlq" {
		t.Fatalf("unexpected author %q", first.Author)
	}
	if first.Replies != 6 {
		t.Fatalf("want 6 replies from href, got %d", first.Replies)
	}
	if first.LastTouchedAt <= 0 {
		t.Fatalf("want parsed last touched time, got %d", first.LastTouchedAt)
	}
	if !strings.Contains(first.Title, "M4 mac mini") {
		t.Fatalf("unexpected title %q", first.Title)
	}
}

func TestParseRecentList(t *testing.T) {
	page := ParseList(loadDoc(t, "r_1.html"), KindRecent)
	if len(page.Entries) != 22 {
		t.Fatalf("want 22 entries, got %d", len(page.Entries))
	}
	if page.Pages != 41731 {
		t.Fatalf("want 41731 pages, got %d", page.Pages)
	}
	if page.Entries[0].ID != 1251929 {
		t.Fatalf("want topic 1251929, got %d", page.Entries[0].ID)
	}
}

func TestParseHomeList(t *testing.T) {
	page := ParseList(loadDoc(t, "home.html"), KindHome)
	if len(page.Entries) != 50 {
		t.Fatalf("want 50 entries, got %d", len(page.Entries))
	}
	if page.Pages != 1 {
		t.Fatalf("want 1 page, got %d", page.Pages)
	}
	first := page.Entries[0]
	if first.ID != 1251873 || first.Node != "programmer" {
		t.Fatalf("unexpected first entry: %+v", first)
	}
	if first.Author != "grandlq" {
		t.Fatalf("unexpected author %q", first.Author)
	}
	if first.Replies != 23 {
		t.Fatalf("want 23 replies, got %d", first.Replies)
	}
}

func TestParseTopicPage(t *testing.T) {
	tp := ParseTopicPage(loadDoc(t, "t1_p1.html"), 1)
	if tp.Topic.ID != 1 {
		t.Fatalf("want id 1, got %d", tp.Topic.ID)
	}
	if !strings.Contains(tp.Topic.Title, "Project Babel") {
		t.Fatalf("unexpected title %q", tp.Topic.Title)
	}
	if tp.Topic.Author != "Livid" {
		t.Fatalf("unexpected author %q", tp.Topic.Author)
	}
	if tp.Topic.Node != "babel" {
		t.Fatalf("unexpected node %q", tp.Topic.Node)
	}
	if tp.Topic.CreatedAt != 1272207387 {
		t.Fatalf("unexpected created_at %d", tp.Topic.CreatedAt)
	}
	if tp.Topic.Replies != 178 {
		t.Fatalf("want 178 replies, got %d", tp.Topic.Replies)
	}
	if tp.Pages != 2 || tp.PageSize != 100 {
		t.Fatalf("unexpected pagination: pages=%d pageSize=%d", tp.Pages, tp.PageSize)
	}
	if len(tp.Replies) != 100 {
		t.Fatalf("want 100 replies, got %d", len(tp.Replies))
	}
	if tp.Replies[0].Floor != 1 || tp.Replies[0].Author != "Jay" {
		t.Fatalf("unexpected first reply: %+v", tp.Replies[0])
	}
	if !strings.Contains(tp.Replies[0].ContentHTML, "又回来了") {
		t.Fatalf("unexpected first reply content %q", tp.Replies[0].ContentHTML)
	}
	if !strings.Contains(tp.Topic.ContentHTML, "Project Babel 2.0 is here") {
		t.Fatalf("unexpected OP content %q", tp.Topic.ContentHTML)
	}
	if tp.Views != 102364 {
		t.Fatalf("want 102364 views, got %d", tp.Views)
	}
}

func TestParseTopicPage2(t *testing.T) {
	tp := ParseTopicPage(loadDoc(t, "t1_p2.html"), 2)
	if tp.Pages != 2 {
		t.Fatalf("want 2 pages, got %d", tp.Pages)
	}
	if len(tp.Replies) != 78 {
		t.Fatalf("want 78 replies, got %d", len(tp.Replies))
	}
	if tp.Replies[0].Floor != 101 {
		t.Fatalf("want first floor 101, got %d", tp.Replies[0].Floor)
	}
	if tp.Replies[len(tp.Replies)-1].Floor != 178 {
		t.Fatalf("want last floor 178, got %d", tp.Replies[len(tp.Replies)-1].Floor)
	}
}

func TestParseJSONFeed(t *testing.T) {
	data, err := os.ReadFile("testdata/feed_tech.json")
	if err != nil {
		t.Fatal(err)
	}
	topics, err := ParseJSONFeed(data, "tech")
	if err != nil {
		t.Fatal(err)
	}
	if len(topics) != 50 {
		t.Fatalf("want 50 topics, got %d", len(topics))
	}
	if topics[0].ID != 1251278 {
		t.Fatalf("want first id 1251278, got %d", topics[0].ID)
	}
	want := time.Date(2026, 10, 23, 7, 17, 21, 0, time.UTC).Unix()
	if topics[0].CreatedAt != want {
		t.Fatalf("want created %d, got %d", want, topics[0].CreatedAt)
	}
	if !strings.Contains(topics[0].ContentHTML, "mac mini") {
		t.Fatalf("unexpected content %q", topics[0].ContentHTML)
	}
	for i := 1; i < len(topics); i++ {
		if topics[i-1].CreatedAt < topics[i].CreatedAt {
			t.Fatalf("feed not sorted at %d", i)
		}
	}
}

func TestParseAtomFeed(t *testing.T) {
	data, err := os.ReadFile("testdata/index.xml")
	if err != nil {
		t.Fatal(err)
	}
	topics, err := ParseAtomFeed(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(topics) != 50 {
		t.Fatalf("want 50 topics, got %d", len(topics))
	}
	if topics[0].NodeTitle != "职场话题" {
		t.Fatalf("unexpected node title %q", topics[0].NodeTitle)
	}
	if !strings.Contains(topics[0].Title, "扛了这么久") {
		t.Fatalf("unexpected title %q", topics[0].Title)
	}
	if topics[0].CreatedAt <= 0 {
		t.Fatalf("missing creation time")
	}
}

func TestParsePlanes(t *testing.T) {
	nodes := ParsePlanes(loadDoc(t, "planes.html"))
	if len(nodes) < 1300 {
		t.Fatalf("want node directory with 1300+ nodes, got %d", len(nodes))
	}
	var tech string
	for _, n := range nodes {
		if n.Name == "tech" {
			tech = n.Title
		}
	}
	if tech != "科技" {
		t.Fatalf("unexpected tech node title %q", tech)
	}
}

func TestSanitizer(t *testing.T) {
	s := NewSanitizer()
	in := `<p>hello<script>alert(1)</script><img src="/static/a.png">` +
		`<a href="javascript:alert(1)">bad</a>` +
		`<iframe src="https://www.youtube.com/embed/abc123"></iframe>` +
		`<iframe src="https://evil.example/embed/x"></iframe></p>`
	out := s.Sanitize(in)
	if strings.Contains(out, "<script") {
		t.Fatalf("script survived: %s", out)
	}
	if strings.Contains(out, "javascript:") {
		t.Fatalf("javascript URL survived: %s", out)
	}
	if !strings.Contains(out, "https://www.v2ex.com/static/a.png") {
		t.Fatalf("relative image not resolved: %s", out)
	}
	if !strings.Contains(out, "youtube.com/embed/abc123") {
		t.Fatalf("youtube iframe removed: %s", out)
	}
	if strings.Count(out, "<iframe") != 1 {
		t.Fatalf("expected exactly one iframe, got: %s", out)
	}
	text := s.Text(in)
	if !strings.Contains(text, "hello") {
		t.Fatalf("unexpected text %q", text)
	}
	if strings.Contains(text, "alert(1)") {
		t.Fatalf("script text leaked: %q", text)
	}
}
