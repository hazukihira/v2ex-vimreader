package service

import (
	"github.com/hazukihira/v2ex-vimreader/internal/model"
	"github.com/hazukihira/v2ex-vimreader/internal/store"
)

// ArchiveInfo describes the creation-time range covered by the local archive.
type ArchiveInfo struct {
	From int64 `json:"from"`
	To   int64 `json:"to"`
}

// BootstrapConfig is the client-safe part of the configuration.
type BootstrapConfig struct {
	AIEnabled bool   `json:"aiEnabled"`
	AIModel   string `json:"aiModel"`
	Theme     string `json:"theme"`
	SiteTitle string `json:"siteTitle"`
	Timezone  string `json:"timezone"`
}

// FetcherStatus reports outbound request statistics.
type FetcherStatus struct {
	Requests      int64  `json:"requests"`
	RateRemaining int    `json:"rateRemaining"`
	LastError     string `json:"lastError"`
}

// SyncStatus reports background sync progress.
type SyncStatus struct {
	LastSiteFeed  int64 `json:"lastSiteFeed"`
	LastNodeFeeds int64 `json:"lastNodeFeeds"`
	Running       bool  `json:"running"`
}

// LoginStatus is the local V2EX login state. The cookie itself is never exposed.
type LoginStatus struct {
	HasCookie bool   `json:"hasCookie"`
	LoggedIn  bool   `json:"loggedIn"`
	User      string `json:"user"`
	CheckedAt int64  `json:"checkedAt"`
}

// StatusResponse is returned by GET /api/status.
type StatusResponse struct {
	DB      store.Stats   `json:"db"`
	Fetcher FetcherStatus `json:"fetcher"`
	Sync    SyncStatus    `json:"sync"`
	Archive ArchiveInfo   `json:"archive"`
	Login   LoginStatus   `json:"login"`
}

// FollowedNodesResponse is returned by GET/PUT /api/followed/nodes.
type FollowedNodesResponse struct {
	FollowedNodes []string     `json:"followedNodes"`
	Nodes         []model.Node `json:"nodes"`
}

// BootstrapResponse is the first call of the web client.
type BootstrapResponse struct {
	Version       string          `json:"version"`
	Nodes         []model.Node    `json:"nodes"`
	FollowedNodes []string        `json:"followedNodes"`
	Config        BootstrapConfig `json:"config"`
	Status        StatusResponse  `json:"status"`
	Archive       ArchiveInfo     `json:"archive"`
	Login         LoginStatus     `json:"login"`
}

// ListResponse is returned by GET /api/list.
type ListResponse struct {
	Kind      string            `json:"kind"`
	Node      string            `json:"node,omitempty"`
	Page      int               `json:"page"`
	Pages     int               `json:"pages"`
	Entries   []model.ListEntry `json:"entries"`
	FetchedAt int64             `json:"fetchedAt"`
	Stale     bool              `json:"stale"`
	Notice    string            `json:"notice,omitempty"`
	Archive   *ArchiveInfo      `json:"archive,omitempty"`
}

// TopicDTO is the topic header shown above the replies.
type TopicDTO struct {
	ID          int64  `json:"id"`
	Node        string `json:"node"`
	NodeTitle   string `json:"nodeTitle"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	CreatedAt   int64  `json:"createdAt"`
	Replies     int    `json:"replies"`
	Views       int64  `json:"views"`
	ContentHTML string `json:"contentHtml"`
	FetchedAt   int64  `json:"fetchedAt"`
}

// ReplyDTO is one floor.
type ReplyDTO struct {
	Floor       int    `json:"floor"`
	Author      string `json:"author"`
	CreatedAt   int64  `json:"createdAt"`
	ContentHTML string `json:"contentHtml"`
	IsOp        bool   `json:"isOp"`
}

// SummaryDTO is the AI summary state of a topic.
type SummaryDTO struct {
	Status    string `json:"status"`
	Content   string `json:"content"`
	Model     string `json:"model"`
	CreatedAt int64  `json:"createdAt"`
}

// TopicResponse is returned by GET /api/topic/{id}.
type TopicResponse struct {
	Topic    TopicDTO   `json:"topic"`
	Page     int        `json:"page"`
	Pages    int        `json:"pages"`
	PageSize int        `json:"pageSize"`
	Replies  []ReplyDTO `json:"replies"`
	Summary  SummaryDTO `json:"summary"`
}

// RepliesResponse is returned by GET /api/topic/{id}/replies.
type RepliesResponse struct {
	Page     int        `json:"page"`
	Pages    int        `json:"pages"`
	PageSize int        `json:"pageSize"`
	Replies  []ReplyDTO `json:"replies"`
}

// BlocksResponse is the local block list.
type BlocksResponse struct {
	Topics []int64  `json:"topics"`
	Users  []string `json:"users"`
}

// NodesResponse is the node directory.
type NodesResponse struct {
	Nodes     []model.Node `json:"nodes"`
	FetchedAt int64        `json:"fetchedAt"`
}

// SearchResponse is returned by GET /api/search.
type SearchResponse struct {
	Topics []model.ListEntry `json:"topics"`
}
