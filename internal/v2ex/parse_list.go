package v2ex

import (
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/hazukihira/v2ex-vimreader/internal/model"
)

// ListKind identifies which V2EX list page is being parsed.
type ListKind string

const (
	KindHome   ListKind = "home"
	KindNode   ListKind = "node"
	KindRecent ListKind = "recent"
)

// ParseList parses a topic list page (home, node or /recent).
func ParseList(doc *goquery.Document, kind ListKind) *model.ListPage {
	entries := make([]model.ListEntry, 0, 50)
	seen := make(map[int64]struct{}, 50)
	scope := listScope(doc, kind)
	links := scope.Find("a.topic-link")
	if links.Length() == 0 {
		links = doc.Find("a.topic-link")
	}
	links.Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		id := topicIDFromHref(href)
		if id == 0 {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		row := a.Closest("div.cell")
		if row.Find("span.topic_info").Length() == 0 && findAttrPrefix(row, "a.node", "href", "/go/") == nil {
			return
		}
		e := model.ListEntry{
			ID:    id,
			Title: strings.TrimSpace(a.Text()),
		}
		if n, ok := replyCountFromHref(href); ok {
			e.Replies = n
		} else {
			e.Replies = 0
		}
		if nodeAnchor := findAttrPrefix(row, "a.node", "href", "/go/"); nodeAnchor != nil {
			nodeHref, _ := nodeAnchor.Attr("href")
			e.Node = strings.TrimPrefix(nodeHref, "/go/")
			e.NodeTitle = strings.TrimSpace(nodeAnchor.Text())
		}
		if authorAnchor := findAttrPrefix(row, "span.topic_info strong a", "href", "/member/"); authorAnchor != nil {
			e.Author = strings.TrimSpace(authorAnchor.Text())
		}
		if clock := row.Find("span.topic_info span").First(); clock.Length() > 0 {
			if title, ok := clock.Attr("title"); ok {
				e.LastTouchedAt = parseUnixTime(title)
			}
		}
		entries = append(entries, e)
	})
	page := &model.ListPage{Entries: entries, Pages: 1, FetchedAt: time.Now().Unix()}
	switch kind {
	case KindNode:
		page.Pages = nodePageCount(doc, len(entries))
	case KindRecent:
		page.Pages = recentPageCount(doc)
	}
	if page.Pages < 1 {
		page.Pages = 1
	}
	return page
}

func listScope(doc *goquery.Document, kind ListKind) *goquery.Selection {
	switch kind {
	case KindNode:
		if sel := doc.Find("#TopicsNode"); sel.Length() > 0 {
			return sel
		}
	case KindHome, KindRecent:
		if sel := doc.Find("#Main"); sel.Length() > 0 {
			return sel
		}
	}
	return doc.Selection
}

func nodePageCount(doc *goquery.Document, rowCount int) int {
	pages := 1
	if m := reNodeTotal.FindStringSubmatch(doc.Text()); m != nil {
		if total := atoiSafe(m[1]); total > 0 && rowCount > 0 {
			pages = (total + rowCount - 1) / rowCount
		}
	}
	if max := maxPageParam(doc); max > pages {
		pages = max
	}
	return pages
}

func recentPageCount(doc *goquery.Document) int {
	pages := 1
	if m := reRecentPages.FindStringSubmatch(doc.Text()); m != nil {
		if v := atoiSafe(m[1]); v > 0 {
			pages = v
		}
	}
	if max := maxPageParam(doc); max > pages {
		pages = max
	}
	return pages
}

func maxPageParam(doc *goquery.Document) int {
	max := 0
	doc.Find("a").Each(func(_ int, a *goquery.Selection) {
		href, ok := a.Attr("href")
		if !ok || !strings.Contains(href, "?p=") {
			return
		}
		u, err := url.Parse(href)
		if err != nil {
			return
		}
		if v := atoiSafe(u.Query().Get("p")); v > max {
			max = v
		}
	})
	return max
}
