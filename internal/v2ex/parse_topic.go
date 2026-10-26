package v2ex

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/hazukihira/v2ex-vimreader/internal/model"
)

const replyPageSize = 100

type ldPosting struct {
	Type          string `json:"@type"`
	Headline      string `json:"headline"`
	Name          string `json:"name"`
	DatePublished string `json:"datePublished"`
	CommentCount  *int   `json:"commentCount"`
	Author        struct {
		Name string `json:"name"`
	} `json:"author"`
}

// ParseTopicPage parses a topic page (up to 100 replies) including the original post.
func ParseTopicPage(doc *goquery.Document, page int) *model.TopicPage {
	tp := &model.TopicPage{Page: page, Pages: 1, PageSize: replyPageSize}
	if page < 1 {
		tp.Page = 1
		page = 1
	}
	if m := reTopicIDScript.FindStringSubmatch(doc.Text()); m != nil {
		tp.Topic.ID = int64(atoiSafe(m[1]))
	}
	var posting ldPosting
	found := false
	doc.Find("script").Each(func(_ int, s *goquery.Selection) {
		if found {
			return
		}
		if typ, _ := s.Attr("type"); typ != "application/ld+json" {
			return
		}
		var candidate ldPosting
		if err := json.Unmarshal([]byte(strings.TrimSpace(s.Text())), &candidate); err != nil {
			return
		}
		if candidate.CommentCount != nil || candidate.DatePublished != "" {
			posting = candidate
			found = true
		}
	})
	if found {
		tp.Topic.Author = posting.Author.Name
		tp.Topic.CreatedAt = parseUnixTime(posting.DatePublished)
		if posting.CommentCount != nil {
			tp.Topic.Replies = *posting.CommentCount
		}
	}
	if tp.Topic.Replies < -1 {
		tp.Topic.Replies = -1
	}
	tp.Topic.Title = strings.TrimSpace(doc.Find("h1").First().Text())
	header := doc.Find("div.header").First()
	if nodeAnchor := findAttrPrefix(header, "a", "href", "/go/"); nodeAnchor != nil {
		if href, ok := nodeAnchor.Attr("href"); ok {
			tp.Topic.Node = strings.TrimPrefix(href, "/go/")
		}
		if tp.Topic.NodeTitle == "" {
			tp.Topic.NodeTitle = strings.TrimSpace(nodeAnchor.Text())
		}
	}
	if tp.Topic.Author == "" {
		if a := findAttrPrefix(header, "a", "href", "/member/"); a != nil {
			tp.Topic.Author = strings.TrimSpace(a.Text())
		}
	}
	if tp.Topic.Replies >= 0 {
		tp.Pages = (tp.Topic.Replies + replyPageSize - 1) / replyPageSize
		if tp.Pages < 1 {
			tp.Pages = 1
		}
	}
	if page > tp.Pages {
		tp.Page = tp.Pages
	}
	if views := reViews.FindStringSubmatch(doc.Find("small.gray").First().Text()); views != nil {
		tp.Views = int64(atoiSafe(views[1]))
		tp.Topic.Views = tp.Views
	}
	if op := doc.Find("div.topic_content").First(); op.Length() > 0 {
		if html, err := op.Html(); err == nil {
			tp.Topic.ContentHTML = html
		}
	}
	doc.Find("div.cell").Each(func(_ int, sel *goquery.Selection) {
		id, hasID := sel.Attr("id")
		if !hasID || !strings.HasPrefix(id, "r_") {
			return
		}
		var r model.Reply
		r.Floor = atoiSafe(sel.Find("span.no").First().Text())
		if r.Floor <= 0 {
			return
		}
		if a := findAttrPrefix(sel, "strong a", "href", "/member/"); a != nil {
			r.Author = strings.TrimSpace(a.Text())
		}
		if clock := sel.Find("span.ago").First(); clock.Length() > 0 {
			if title, ok := clock.Attr("title"); ok {
				r.CreatedAt = parseUnixTime(title)
			}
		}
		if body := sel.Find("div.reply_content").First(); body.Length() > 0 {
			if html, err := body.Html(); err == nil {
				r.ContentHTML = html
			}
		}
		r.FetchedAt = time.Now().Unix()
		tp.Replies = append(tp.Replies, r)
	})
	return tp
}
