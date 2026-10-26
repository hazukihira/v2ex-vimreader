// Package model holds the shared data types used by the store, the V2EX
// parsers, the service layer and the HTTP API.
package model

// Node is a V2EX node (论坛节点).
type Node struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	TitleAlt    string `json:"titleAlt"`
	Parent      string `json:"parent"`
	TopicsCount int    `json:"topicsCount"`
	Restricted  bool   `json:"restricted"`
}

// Topic is a forum topic. Replies is -1 when the reply count is not known yet.
type Topic struct {
	ID            int64  `json:"id"`
	Node          string `json:"node"`
	NodeTitle     string `json:"nodeTitle"`
	Title         string `json:"title"`
	Author        string `json:"author"`
	CreatedAt     int64  `json:"createdAt"`
	LastTouchedAt int64  `json:"lastTouchedAt"`
	Replies       int    `json:"replies"`
	Views         int64  `json:"views"`
	ContentHTML   string `json:"-"`
	ContentText   string `json:"-"`
	FetchedAt     int64  `json:"fetchedAt"`
	FeedAt        int64  `json:"-"`
}

// Reply is one floor (楼层) of a topic.
type Reply struct {
	TopicID     int64  `json:"-"`
	Floor       int    `json:"floor"`
	Author      string `json:"author"`
	CreatedAt   int64  `json:"createdAt"`
	ContentHTML string `json:"contentHtml"`
	ContentText string `json:"-"`
	FetchedAt   int64  `json:"-"`
}

// ListEntry is one row of a topic list page.
type ListEntry struct {
	ID            int64  `json:"id"`
	Node          string `json:"node"`
	NodeTitle     string `json:"nodeTitle"`
	Title         string `json:"title"`
	Author        string `json:"author"`
	Replies       int    `json:"replies"`
	CreatedAt     int64  `json:"createdAt"`
	CreatedKnown  bool   `json:"createdKnown"`
	LastTouchedAt int64  `json:"lastTouchedAt"`
	Pinned        bool   `json:"pinned"`
}

// ListPage is a parsed site list page.
type ListPage struct {
	Entries   []ListEntry
	Pages     int
	FetchedAt int64
	Notice    string
}

// TopicPage is a parsed topic page: topic header + one page of replies.
type TopicPage struct {
	Topic    Topic
	Replies  []Reply
	Page     int
	Pages    int
	PageSize int
	Views    int64
}
