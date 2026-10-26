package v2ex

import (
	"encoding/json"
	"encoding/xml"
	"strconv"
	"strings"
	"time"

	"github.com/hazukihira/v2ex-vimreader/internal/model"
)

type jsonFeed struct {
	Title string `json:"title"`
	Items []struct {
		ID            string `json:"id"`
		URL           string `json:"url"`
		Title         string `json:"title"`
		DatePublished string `json:"date_published"`
		ContentHTML   string `json:"content_html"`
		Author        struct {
			Name string `json:"name"`
		} `json:"author"`
	} `json:"items"`
}

// ParseJSONFeed parses /feed/<node>.json (50 newest topics of one node, created desc).
func ParseJSONFeed(data []byte, nodeName string) ([]model.Topic, error) {
	var f jsonFeed
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	out := make([]model.Topic, 0, len(f.Items))
	for _, item := range f.Items {
		idText := item.ID
		if m := reTopicPath.FindStringSubmatch(item.URL); m != nil {
			idText = m[1]
		}
		id, err := strconv.ParseInt(strings.TrimSpace(idText), 10, 64)
		if err != nil || id == 0 {
			continue
		}
		out = append(out, model.Topic{
			ID:          id,
			Node:        nodeName,
			Title:       strings.TrimSpace(item.Title),
			Author:      item.Author.Name,
			CreatedAt:   parseUnixTime(item.DatePublished),
			Replies:     -1,
			ContentHTML: item.ContentHTML,
			FeedAt:      now,
		})
	}
	return out, nil
}

type atomFeed struct {
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title     string     `xml:"title"`
	Links     []atomLink `xml:"link"`
	ID        string     `xml:"id"`
	Published string     `xml:"published"`
	Updated   string     `xml:"updated"`
	Author    struct {
		Name string `xml:"name"`
	} `xml:"author"`
	Content string `xml:"content"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

// ParseAtomFeed parses /index.xml or /read.xml (50 newest site-wide topics, created desc).
func ParseAtomFeed(data []byte) ([]model.Topic, error) {
	var f atomFeed
	if err := xml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	out := make([]model.Topic, 0, len(f.Entries))
	for _, e := range f.Entries {
		link := ""
		for _, l := range e.Links {
			if l.Rel == "" || l.Rel == "alternate" {
				link = l.Href
				break
			}
		}
		if link == "" {
			link = e.ID
		}
		id := topicIDFromHref(link)
		if id == 0 {
			continue
		}
		title := strings.TrimSpace(e.Title)
		nodeTitle := ""
		if m := reNodePrefix.FindStringSubmatch(title); m != nil {
			nodeTitle = strings.TrimSpace(m[1])
			title = strings.TrimSpace(m[2])
		}
		replies, _ := replyCountFromHref(link)
		created := parseUnixTime(e.Published)
		if created == 0 {
			created = parseUnixTime(e.Updated)
		}
		out = append(out, model.Topic{
			ID:          id,
			NodeTitle:   nodeTitle,
			Title:       title,
			Author:      e.Author.Name,
			CreatedAt:   created,
			Replies:     replies,
			ContentHTML: e.Content,
			FeedAt:      now,
		})
	}
	return out, nil
}
