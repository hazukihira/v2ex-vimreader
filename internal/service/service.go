// Package service coordinates the V2EX client, the SQLite cache and the
// background incremental sync.
package service

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hazukihira/v2ex-vimreader/internal/ai"
	"github.com/hazukihira/v2ex-vimreader/internal/config"
	"github.com/hazukihira/v2ex-vimreader/internal/model"
	"github.com/hazukihira/v2ex-vimreader/internal/store"
	"github.com/hazukihira/v2ex-vimreader/internal/v2ex"
)

const (
	archivePageSize  = 50
	nodeDirectoryTTL = 7 * 24 * time.Hour
)

// Service is the application core.
type Service struct {
	cfg *config.Config
	st  *store.Store
	vc  *v2ex.Client
	san *v2ex.Sanitizer
	ai  *ai.Client
	log *slog.Logger
	loc *time.Location

	flights flightGroup

	stateMu       sync.Mutex
	syncing       bool
	lastSiteFeed  int64
	lastNodeFeeds int64
	lastError     string

	restricted map[string]struct{}

	loginMu        sync.RWMutex
	loginCookie    string
	loginLoggedIn  bool
	loginUser      string
	loginCheckedAt int64
	loginChecked   bool

	followedMu sync.RWMutex
	followed   []string

	cancel context.CancelFunc
	wg     sync.WaitGroup

	prefetchMu     sync.Mutex
	prefetchFailed map[int64]time.Time
}

// New creates the service.
func New(cfg *config.Config, st *store.Store, log *slog.Logger) *Service {
	restricted := make(map[string]struct{})
	for _, name := range cfg.Sync.RestrictedNodes {
		name = strings.ToLower(strings.TrimSpace(name))
		if name != "" {
			restricted[name] = struct{}{}
		}
	}
	svc := &Service{
		cfg:        cfg,
		st:         st,
		vc:         v2ex.NewClient(cfg.Fetch, log),
		san:        v2ex.NewSanitizer(),
		ai:         ai.New(cfg.AI, cfg.AIAPIKey()),
		log:        log,
		loc:        cfg.Location(),
		restricted: restricted,
	}
	svc.loadLogin()
	svc.loadFollowedNodes()
	return svc
}

// Config exposes the configuration.
func (s *Service) Config() *config.Config { return s.cfg }

// Store exposes the database for tests and extensions.
func (s *Service) Store() *store.Store { return s.st }

// SetError records the last outbound error for the status bar.
func (s *Service) SetError(err error) {
	if err == nil {
		return
	}
	s.stateMu.Lock()
	s.lastError = err.Error()
	s.stateMu.Unlock()
}

// Bootstrap returns the first-screen payload.
func (s *Service) Bootstrap(ctx context.Context) (*BootstrapResponse, error) {
	names := s.FollowedNodeNames()
	nodes := make([]model.Node, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		n, err := s.st.GetNode(ctx, name)
		if err != nil {
			n = model.Node{Name: name, Title: name}
		}
		nodes = append(nodes, n)
	}
	status, err := s.Status(ctx)
	if err != nil {
		return nil, err
	}
	from, to, err := s.st.CoverageRange(ctx)
	if err != nil {
		return nil, err
	}
	return &BootstrapResponse{
		Version:       config.Version,
		Nodes:         nodes,
		FollowedNodes: names,
		Config: BootstrapConfig{
			AIEnabled: s.ai.Enabled(),
			AIModel:   s.ai.Model(),
			Theme:     "dark",
			SiteTitle: "v2ex-vimreader",
			Timezone:  s.cfg.Timezone,
		},
		Status:  *status,
		Archive: ArchiveInfo{From: from, To: to},
		Login:   s.LoginStatus(),
	}, nil
}

// Status reports local cache size, fetcher and sync state.
func (s *Service) Status(ctx context.Context) (*StatusResponse, error) {
	stats, err := s.st.Stats(ctx)
	if err != nil {
		return nil, err
	}
	from, to, err := s.st.CoverageRange(ctx)
	if err != nil {
		return nil, err
	}
	s.stateMu.Lock()
	running := s.syncing
	lastSite := s.lastSiteFeed
	lastNode := s.lastNodeFeeds
	lastErr := s.lastError
	s.stateMu.Unlock()
	var requests int64
	if limiter := s.vc.Limiter(); limiter != nil {
		requests, _ = limiter.Stats()
	}
	return &StatusResponse{
		DB:    stats,
		Login: s.LoginStatus(),
		Fetcher: FetcherStatus{
			Requests:      requests,
			RateRemaining: s.vc.Remaining(),
			LastError:     lastErr,
		},
		Sync: SyncStatus{
			LastSiteFeed:  lastSite,
			LastNodeFeeds: lastNode,
			Running:       running,
		},
		Archive: ArchiveInfo{From: from, To: to},
	}, nil
}

// StartBackground launches the incremental sync loops.
func (s *Service) StartBackground() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.wg.Add(1)
	go func() { defer s.wg.Done(); s.cacheMaintenanceLoop(ctx) }()
	if !s.cfg.Sync.Enabled {
		return
	}
	s.wg.Add(4)
	go func() { defer s.wg.Done(); s.nodeDirectoryLoop(ctx) }()
	go func() { defer s.wg.Done(); s.siteFeedLoop(ctx) }()
	go func() { defer s.wg.Done(); s.nodeFeedsLoop(ctx) }()
	go func() { defer s.wg.Done(); s.loginLoop(ctx) }()
}

// Stop cancels background work and waits for it.
func (s *Service) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

// TriggerSync starts a manual sync in the background. It reports whether the
// sync actually started (a sync already running returns false).
func (s *Service) TriggerSync(target string) (bool, error) {
	switch target {
	case "", "all", "site", "followed":
	default:
		return false, ErrBadRequest
	}
	s.stateMu.Lock()
	if s.syncing {
		s.stateMu.Unlock()
		return false, nil
	}
	s.syncing = true
	s.stateMu.Unlock()
	go func() {
		defer func() {
			s.stateMu.Lock()
			s.syncing = false
			s.stateMu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if target == "" || target == "all" || target == "site" {
			if err := s.syncSiteFeed(ctx); err != nil {
				s.SetError(err)
				s.log.Warn("site feed sync failed", "err", err)
			}
		}
		if target == "" || target == "all" || target == "followed" {
			for _, node := range s.FollowedNodeNames() {
				if err := s.syncNodeFeed(ctx, node); err != nil {
					s.SetError(err)
					s.log.Warn("node feed sync failed", "node", node, "err", err)
				}
			}
			s.markNodeFeedsDone()
		}
	}()
	return true, nil
}

func (s *Service) markSiteFeedDone() {
	s.stateMu.Lock()
	s.lastSiteFeed = time.Now().Unix()
	s.stateMu.Unlock()
}

func (s *Service) markNodeFeedsDone() {
	s.stateMu.Lock()
	s.lastNodeFeeds = time.Now().Unix()
	s.stateMu.Unlock()
}

func (s *Service) metaInt(ctx context.Context, key string) int64 {
	v, err := s.st.GetMeta(ctx, key)
	if err != nil {
		return 0
	}
	n, _ := strconv.ParseInt(v, 10, 64)
	return n
}

func (s *Service) metaSetInt(ctx context.Context, key string, value int64) {
	if err := s.st.SetMeta(ctx, key, strconv.FormatInt(value, 10)); err != nil {
		s.log.Warn("set meta failed", "key", key, "err", err)
	}
}

func isFresh(stamp int64, ttl time.Duration) bool {
	if stamp <= 0 {
		return false
	}
	return time.Since(time.Unix(stamp, 0)) < ttl
}

// flightGroup deduplicates identical concurrent fetches.
type flightGroup struct {
	mu sync.Mutex
	m  map[string]*flightCall
}

type flightCall struct {
	done chan struct{}
	val  any
	err  error
}

func (g *flightGroup) Do(key string, fn func() (any, error)) (any, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*flightCall)
	}
	if call, ok := g.m[key]; ok {
		g.mu.Unlock()
		<-call.done
		return call.val, call.err
	}
	call := &flightCall{done: make(chan struct{})}
	g.m[key] = call
	g.mu.Unlock()

	call.val, call.err = fn()
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
	close(call.done)
	return call.val, call.err
}
