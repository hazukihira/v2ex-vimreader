package service

import (
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/hazukihira/v2ex-vimreader/internal/model"
	"github.com/hazukihira/v2ex-vimreader/internal/store"
	"github.com/hazukihira/v2ex-vimreader/internal/v2ex"
)

const deeperListTTL = 10 * time.Minute

// List returns one page of a site list or of the local archive.
func (s *Service) List(ctx context.Context, kind, node string, page int, date string, refresh bool) (*ListResponse, error) {
	s.MarkUserActivity()
	kind = strings.ToLower(strings.TrimSpace(kind))
	if page < 1 {
		page = 1
	}
	switch kind {
	case "home", "recent", "node":
		return s.siteList(ctx, kind, strings.TrimSpace(node), page, refresh)
	case "new":
		return s.archiveList(ctx, page)
	case "day":
		return s.dayList(ctx, strings.TrimSpace(date), page)
	default:
		return nil, ErrBadRequest
	}
}

// homeArchiveCutoff returns the oldest activity time on the live home page.
func homeArchiveCutoff(entries []model.ListEntry) int64 {
	var cutoff int64
	for _, entry := range entries {
		at := entry.LastTouchedAt
		if at == 0 {
			at = entry.CreatedAt
		}
		if at == 0 {
			continue
		}
		if cutoff == 0 || at < cutoff {
			cutoff = at
		}
	}
	return cutoff
}

func (s *Service) homeArchivePages(ctx context.Context, entries []model.ListEntry) (int, int64, error) {
	cutoff := homeArchiveCutoff(entries)
	if cutoff == 0 {
		return 1, 0, nil
	}
	total, err := s.st.CountActiveBefore(ctx, cutoff)
	if err != nil {
		return 1, 0, err
	}
	if total == 0 {
		return 1, cutoff, nil
	}
	return 1 + (total+archivePageSize-1)/archivePageSize, cutoff, nil
}

func (s *Service) homeArchivePage(ctx context.Context, page int) (*ListResponse, error) {
	cached, ok, err := s.st.GetListPage(ctx, "home", 1)
	if err != nil {
		return nil, err
	}
	if !ok {
		topics, total, err := s.st.ListNew(ctx, page-1, archivePageSize)
		if err != nil {
			return nil, err
		}
		entries := make([]model.ListEntry, 0, len(topics))
		for _, topic := range topics {
			entries = append(entries, TopicToListEntry(topic))
		}
		pages := 1 + (total+archivePageSize-1)/archivePageSize
		if pages < 1 {
			pages = 1
		}
		if page > pages {
			page = pages
		}
		return s.buildListResponse(ctx, "home", "", page, model.ListPage{Entries: entries, Pages: pages, FetchedAt: time.Now().Unix()}, true, "")
	}
	pages, cutoff, err := s.homeArchivePages(ctx, cached.Entries)
	if err != nil {
		return nil, err
	}
	if page > pages {
		page = pages
	}
	topics, _, err := s.st.ListActiveBefore(ctx, cutoff, page-1, archivePageSize)
	if err != nil {
		return nil, err
	}
	entries := make([]model.ListEntry, 0, len(topics))
	var fetchedAt int64
	for _, topic := range topics {
		entries = append(entries, TopicToListEntry(topic))
		if topic.LastTouchedAt > fetchedAt {
			fetchedAt = topic.LastTouchedAt
		}
		if topic.CreatedAt > fetchedAt {
			fetchedAt = topic.CreatedAt
		}
	}
	if fetchedAt == 0 {
		fetchedAt = time.Now().Unix()
	}
	return s.buildListResponse(ctx, "home", "", page, model.ListPage{Entries: entries, Pages: pages, FetchedAt: fetchedAt}, true, "")
}

func (s *Service) siteList(ctx context.Context, kind, node string, page int, refresh bool) (*ListResponse, error) {
	if kind == "node" && node == "" {
		return nil, ErrBadRequest
	}
	if kind == "home" && page > 1 {
		return s.homeArchivePage(ctx, page)
	}
	var key, rawURL string
	var listKind v2ex.ListKind
	switch kind {
	case "home":
		key, rawURL, listKind = "home", v2ex.HomeURL(), v2ex.KindHome
	case "recent":
		key, rawURL, listKind = "recent", v2ex.RecentURL(page), v2ex.KindRecent
	case "node":
		key, rawURL, listKind = "node:"+node, v2ex.NodeURL(node, page), v2ex.KindNode
	}
	ttl := s.cfg.Fetch.ListCacheTTL()
	if page > 1 && ttl < deeperListTTL {
		ttl = deeperListTTL
	}
	if !refresh {
		if cached, ok, err := s.st.GetListPage(ctx, key, page); err != nil {
			return nil, err
		} else if ok && isFresh(cached.FetchedAt, ttl) {
			if kind == "home" {
				if pages, _, err := s.homeArchivePages(ctx, cached.Entries); err == nil && pages > cached.Pages {
					cached.Pages = pages
				}
			}
			return s.buildListResponse(ctx, kind, node, page, cached, false, "")
		}
	}
	if kind == "node" && !s.loggedIn() {
		if n, err := s.st.GetNode(ctx, node); err == nil && n.Restricted {
			return s.restrictedNodeResponse(kind, node), nil
		}
	}
	if kind == "node" && !refresh {
		if local, ok, err := s.nodeArchiveResponse(ctx, node, page); err != nil {
			return nil, err
		} else if ok {
			return local, nil
		}
	}
	resp, err := s.vc.Get(ctx, rawURL)
	if err != nil {
		s.SetError(err)
		return nil, err
	}
	if resp.IsLoginWall() {
		if kind == "node" {
			_ = s.st.SetNodeRestricted(ctx, node, true)
		}
		return s.restrictedNodeResponse(kind, node), nil
	}
	if err := resp.Err(); err != nil {
		s.SetError(err)
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(resp.Body))
	if err != nil {
		return nil, err
	}
	parsed := v2ex.ParseList(doc, listKind)
	if kind == "home" {
		if pages, _, err := s.homeArchivePages(ctx, parsed.Entries); err == nil && pages > parsed.Pages {
			parsed.Pages = pages
		}
	}
	if kind == "node" {
		title := s.nodeTitle(ctx, node)
		for i := range parsed.Entries {
			parsed.Entries[i].Node = node
			if parsed.Entries[i].NodeTitle == "" {
				parsed.Entries[i].NodeTitle = title
			}
		}
	}
	for _, entry := range parsed.Entries {
		topic := model.Topic{
			ID:            entry.ID,
			Node:          entry.Node,
			NodeTitle:     entry.NodeTitle,
			Title:         entry.Title,
			Author:        entry.Author,
			LastTouchedAt: entry.LastTouchedAt,
			Replies:       entry.Replies,
		}
		if err := s.st.UpsertTopic(ctx, topic); err != nil {
			s.log.Warn("upsert list topic failed", "id", entry.ID, "err", err)
		}
	}
	if err := s.st.SaveListPage(ctx, key, page, parsed.Pages, parsed.Entries); err != nil {
		s.log.Warn("save list page failed", "key", key, "page", page, "err", err)
	}
	return s.buildListResponse(ctx, kind, node, page, *parsed, false, "")
}

// nodeArchiveResponse serves a node page from the local archive. Node list
// pages are the main navigation path; waiting for V2EX on every click makes
// the sidebar feel stuck, while the feed sync already keeps recent topics here.
func (s *Service) nodeArchiveResponse(ctx context.Context, node string, page int) (*ListResponse, bool, error) {
	const pageSize = 50
	topics, total, err := s.st.ListNode(ctx, node, page, pageSize)
	if err != nil {
		return nil, false, err
	}
	if total == 0 {
		return nil, false, nil
	}
	pages := (total + pageSize - 1) / pageSize
	if pages < 1 {
		pages = 1
	}
	entries := make([]model.ListEntry, 0, len(topics))
	var fetchedAt int64
	for _, topic := range topics {
		entries = append(entries, TopicToListEntry(topic))
		if topic.LastTouchedAt > fetchedAt {
			fetchedAt = topic.LastTouchedAt
		}
		if topic.CreatedAt > fetchedAt {
			fetchedAt = topic.CreatedAt
		}
	}
	if fetchedAt == 0 {
		fetchedAt = time.Now().Unix()
	}
	resp, err := s.buildListResponse(ctx, "node", node, page, model.ListPage{
		Entries: entries, Pages: pages, FetchedAt: fetchedAt,
	}, true, "")
	if err != nil {
		return nil, false, err
	}
	return resp, true, nil
}

func (s *Service) restrictedNodeResponse(kind, node string) *ListResponse {
	return &ListResponse{
		Kind:      kind,
		Node:      node,
		Page:      1,
		Pages:     1,
		Entries:   []model.ListEntry{},
		FetchedAt: time.Now().Unix(),
		Notice:    "该节点需要登录 V2EX 后才能查看",
	}
}

func (s *Service) archiveList(ctx context.Context, page int) (*ListResponse, error) {
	topics, total, err := s.st.ListNew(ctx, page, archivePageSize)
	if err != nil {
		return nil, err
	}
	entries := make([]model.ListEntry, 0, len(topics))
	for _, topic := range topics {
		entries = append(entries, TopicToListEntry(topic))
	}
	pages := (total + archivePageSize - 1) / archivePageSize
	if pages < 1 {
		pages = 1
	}
	return s.buildListResponse(ctx, "new", "", page, model.ListPage{
		Entries: entries, Pages: pages, FetchedAt: time.Now().Unix(),
	}, false, "")
}

func (s *Service) dayList(ctx context.Context, date string, page int) (*ListResponse, error) {
	if date == "" {
		date = time.Now().In(s.loc).Format("2006-01-02")
	}
	start, end, err := store.DayRange(date, s.loc)
	if err != nil {
		return nil, ErrBadRequest
	}
	topics, total, err := s.st.ListDay(ctx, start, end, page, archivePageSize)
	if err != nil {
		return nil, err
	}
	from, to, err := s.st.CoverageRange(ctx)
	if err != nil {
		return nil, err
	}
	entries := make([]model.ListEntry, 0, len(topics))
	for _, topic := range topics {
		entries = append(entries, TopicToListEntry(topic))
	}
	pages := (total + archivePageSize - 1) / archivePageSize
	if pages < 1 {
		pages = 1
	}
	notice := ""
	if total == 0 {
		switch {
		case from == 0:
			notice = "本地归档还没有数据，后台同步完成后即可按天浏览"
		case end <= from:
			notice = "该日期早于本地归档覆盖范围"
		case start > to:
			notice = "该日期暂无数据；如果选的是今天，请等后台同步或手动刷新"
		default:
			notice = "该日期处于归档缺口，可等待后台同步补全新主题"
		}
	}
	resp, err := s.buildListResponse(ctx, "day", "", page, model.ListPage{
		Entries: entries, Pages: pages, FetchedAt: time.Now().Unix(),
	}, false, notice)
	if err != nil {
		return nil, err
	}
	resp.Archive = &ArchiveInfo{From: from, To: to}
	return resp, nil
}

func (s *Service) buildListResponse(ctx context.Context, kind, node string, page int, parsed model.ListPage, stale bool, notice string) (*ListResponse, error) {
	blockedTopics, blockedUsers, err := s.blockedSets(ctx)
	if err != nil {
		return nil, err
	}
	entries := make([]model.ListEntry, 0, len(parsed.Entries))
	for _, entry := range parsed.Entries {
		if blockedTopics[entry.ID] || blockedUsers[entry.Author] {
			continue
		}
		if entry.Node == "" && kind == "node" {
			entry.Node = node
		}
		if entry.NodeTitle == "" && entry.Node != "" {
			entry.NodeTitle = s.nodeTitle(ctx, entry.Node)
		}
		entry.CreatedKnown = entry.CreatedAt > 0
		entries = append(entries, entry)
	}
	if entries == nil {
		entries = []model.ListEntry{}
	}
	return &ListResponse{
		Kind:      kind,
		Node:      node,
		Page:      page,
		Pages:     parsed.Pages,
		Entries:   entries,
		FetchedAt: parsed.FetchedAt,
		Stale:     stale,
		Notice:    notice,
	}, nil
}

// TopicToListEntry converts a stored topic to a list row.
func TopicToListEntry(topic model.Topic) model.ListEntry {
	return model.ListEntry{
		ID:            topic.ID,
		Node:          topic.Node,
		NodeTitle:     topic.NodeTitle,
		Title:         topic.Title,
		Author:        topic.Author,
		Replies:       topic.Replies,
		CreatedAt:     topic.CreatedAt,
		CreatedKnown:  topic.CreatedAt > 0,
		LastTouchedAt: topic.LastTouchedAt,
	}
}
