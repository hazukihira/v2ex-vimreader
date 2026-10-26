// Package api exposes the HTTP JSON API and the embedded web client.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hazukihira/v2ex-vimreader/internal/ai"
	"github.com/hazukihira/v2ex-vimreader/internal/service"
	"github.com/hazukihira/v2ex-vimreader/internal/store"
	"github.com/hazukihira/v2ex-vimreader/internal/v2ex"
)

// Server wires the service to HTTP routes.
type Server struct {
	svc *service.Service
	web fs.FS
	log *slog.Logger
	mux *http.ServeMux
	dev bool
}

// Option customises the server.
type Option func(*Server)

// WithDevMode disables static asset caching (frontend read from disk).
func WithDevMode() Option { return func(s *Server) { s.dev = true } }

// NewServer creates the HTTP server. web is the embedded web directory.
func NewServer(svc *service.Service, web fs.FS, log *slog.Logger, opts ...Option) *Server {
	s := &Server{svc: svc, web: web, log: log, mux: http.NewServeMux()}
	for _, opt := range opts {
		opt(s)
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/bootstrap", s.handleBootstrap)
	s.mux.HandleFunc("GET /api/list", s.handleList)
	s.mux.HandleFunc("GET /api/topic/{id}", s.handleTopic)
	s.mux.HandleFunc("GET /api/topic/{id}/replies", s.handleReplies)
	s.mux.HandleFunc("POST /api/topic/{id}/refresh", s.handleTopicRefresh)
	s.mux.HandleFunc("GET /api/topic/{id}/summary", s.handleSummaryGet)
	s.mux.HandleFunc("POST /api/topic/{id}/summary", s.handleSummaryPost)
	s.mux.HandleFunc("GET /api/blocks", s.handleBlocksGet)
	s.mux.HandleFunc("POST /api/blocks", s.handleBlocksAdd)
	s.mux.HandleFunc("DELETE /api/blocks", s.handleBlocksRemove)
	s.mux.HandleFunc("GET /api/nodes", s.handleNodes)
	s.mux.HandleFunc("GET /api/status", s.handleStatus)
	s.mux.HandleFunc("POST /api/sync", s.handleSync)
	s.mux.HandleFunc("GET /api/search", s.handleSearch)
	s.mux.HandleFunc("POST /api/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/login/chrome", s.handleLoginChrome)
	s.mux.HandleFunc("GET /api/followed/nodes", s.handleFollowedNodesGet)
	s.mux.HandleFunc("PUT /api/followed/nodes", s.handleFollowedNodesPut)
	s.mux.HandleFunc("DELETE /api/login", s.handleLogout)
	s.mux.HandleFunc("/", s.handleStatic)
}

// ServeHTTP implements http.Handler with panic recovery and request logging.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		if rec := recover(); rec != nil {
			s.log.Error("panic while serving request", "method", r.Method, "path", r.URL.Path, "panic", rec)
			s.fail(w, fmt.Errorf("internal panic"))
		}
	}()
	s.mux.ServeHTTP(w, r)
	s.log.Debug("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(value)
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]apiError{"error": {Code: code, Message: message}})
}

func (s *Server) fail(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	message := "internal server error"
	switch {
	case errors.Is(err, service.ErrBadRequest):
		status, code, message = http.StatusBadRequest, "bad_request", "invalid request parameters"
	case errors.Is(err, service.ErrNotFound), errors.Is(err, store.ErrNotFound):
		status, code, message = http.StatusNotFound, "not_found", "not found"
	case errors.Is(err, service.ErrLoginFailed):
		status, code, message = http.StatusBadRequest, "login_failed", err.Error()
	case errors.Is(err, service.ErrChromeUnavailable):
		status, code, message = http.StatusBadGateway, "chrome_unavailable", err.Error()
	case errors.Is(err, service.ErrInvalidNode):
		status, code, message = http.StatusBadRequest, "invalid_node", err.Error()
	case errors.Is(err, service.ErrLoginRequired):
		status, code, message = http.StatusForbidden, "login_required", err.Error()
	case errors.Is(err, ai.ErrDisabled):
		status, code, message = http.StatusNotImplemented, "ai_disabled", "AI summary is not configured"
	case errors.Is(err, service.ErrUnavailable):
		status, code, message = http.StatusBadGateway, "upstream_error", "V2EX is temporarily unavailable"
	default:
		var httpErr *v2ex.HTTPError
		var urlErr *url.Error
		switch {
		case errors.As(err, &httpErr):
			status, code, message = http.StatusBadGateway, "upstream_error", httpErr.Error()
		case errors.As(err, &urlErr):
			status, code, message = http.StatusBadGateway, "upstream_error", "request to V2EX failed: "+urlErr.Error()
		case errors.Is(err, context.DeadlineExceeded):
			status, code, message = http.StatusGatewayTimeout, "timeout", "request timed out"
		case errors.Is(err, context.Canceled):
			status, code, message = 499, "canceled", "request canceled"
		default:
			s.log.Error("api error", "err", err)
		}
	}
	writeAPIError(w, status, code, message)
}

func queryInt(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

func queryBool(r *http.Request, key string) bool {
	v := strings.ToLower(strings.TrimSpace(r.URL.Query().Get(key)))
	return v == "1" || v == "true" || v == "yes"
}

func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, service.ErrBadRequest
	}
	return id, nil
}

func (s *Server) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	resp, err := s.svc.Bootstrap(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = "home"
	}
	resp, err := s.svc.List(r.Context(), kind,
		r.URL.Query().Get("node"),
		queryInt(r, "page", 1),
		r.URL.Query().Get("date"),
		queryBool(r, "refresh"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleTopic(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	resp, err := s.svc.Topic(r.Context(), id, queryInt(r, "page", 1), queryBool(r, "refresh"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleReplies(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	resp, err := s.svc.Replies(r.Context(), id, queryInt(r, "page", 1), queryBool(r, "refresh"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleTopicRefresh(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	resp, err := s.svc.RefreshTopic(r.Context(), id, queryInt(r, "page", 1))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSummaryGet(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	resp, err := s.svc.Summary(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSummaryPost(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	refresh := queryBool(r, "refresh")
	if r.Body != nil {
		var body struct {
			Refresh bool `json:"refresh"`
		}
		if data, readErr := io.ReadAll(io.LimitReader(r.Body, 4096)); readErr == nil && len(data) > 0 {
			if json.Unmarshal(data, &body) == nil && body.Refresh {
				refresh = true
			}
		}
	}
	if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		s.streamSummary(w, r, id, refresh)
		return
	}
	resp, err := s.svc.GenerateSummary(r.Context(), id, refresh, nil)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) streamSummary(w http.ResponseWriter, r *http.Request, id int64, refresh bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.fail(w, errors.New("streaming unsupported"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	send := func(payload any) error {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}
	onDelta := func(chunk string) error {
		return send(map[string]string{"delta": chunk})
	}
	resp, err := s.svc.GenerateSummary(r.Context(), id, refresh, onDelta)
	if err != nil {
		_ = send(map[string]string{"error": err.Error()})
		return
	}
	_ = send(map[string]any{"done": true, "model": resp.Model})
}

func (s *Server) handleBlocksGet(w http.ResponseWriter, r *http.Request) {
	resp, err := s.svc.Blocks(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleBlocksAdd(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Kind  string          `json:"kind"`
		Value json.RawMessage `json:"value"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil {
		s.fail(w, service.ErrBadRequest)
		return
	}
	value := strings.TrimSpace(string(body.Value))
	var num int64
	if err := json.Unmarshal(body.Value, &num); err == nil {
		value = strconv.FormatInt(num, 10)
	} else if err := json.Unmarshal(body.Value, new(string)); err == nil {
		var str string
		_ = json.Unmarshal(body.Value, &str)
		value = str
	}
	if err := s.svc.AddBlock(r.Context(), body.Kind, value); err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleBlocksRemove(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.RemoveBlock(r.Context(), r.URL.Query().Get("kind"), r.URL.Query().Get("value")); err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	resp, err := s.svc.Nodes(r.Context(), queryBool(r, "refresh"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	resp, err := s.svc.Status(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Target string `json:"target"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body)
	}
	if body.Target == "" {
		body.Target = r.URL.Query().Get("target")
	}
	started, err := s.svc.TriggerSync(body.Target)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"started": started})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	resp, err := s.svc.Search(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Cookie string `json:"cookie"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 64*1024)).Decode(&body); err != nil {
		s.fail(w, service.ErrBadRequest)
		return
	}
	if strings.TrimSpace(body.Cookie) == "" {
		s.fail(w, service.ErrBadRequest)
		return
	}
	status, err := s.svc.SetLoginCookie(r.Context(), body.Cookie)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleFollowedNodesGet(w http.ResponseWriter, r *http.Request) {
	names, nodes, err := s.svc.FollowedNodeInfos(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, service.FollowedNodesResponse{FollowedNodes: names, Nodes: nodes})
}

func (s *Server) handleFollowedNodesPut(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Nodes []string `json:"nodes"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 16*1024)).Decode(&body); err != nil {
		s.fail(w, service.ErrBadRequest)
		return
	}
	names, nodes, err := s.svc.SetFollowedNodes(r.Context(), body.Nodes)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, service.FollowedNodesResponse{FollowedNodes: names, Nodes: nodes})
}

func (s *Server) handleLoginChrome(w http.ResponseWriter, r *http.Request) {
	if !isLoopbackRequest(r) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "只允许从本机导入 Chrome Cookie")
		return
	}
	status, err := s.svc.ImportChromeCookie(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func isLoopbackRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	status, err := s.svc.ClearLoginCookie(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeAPIError(w, http.StatusNotFound, "not_found", "unknown API endpoint")
		return
	}
	clean := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if clean == "" || clean == "." {
		clean = "index.html"
	}
	f, err := s.web.Open(clean)
	if err != nil {
		s.serveIndex(w, r)
		return
	}
	f.Close()
	// Static assets are local and unhashed; never let the browser keep a stale
	// module graph after the binary (or a dev edit) changes.
	w.Header().Set("Cache-Control", "no-store")
	http.FileServer(http.FS(s.web)).ServeHTTP(w, r)
}

func (s *Server) serveIndex(w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(s.web, "index.html")
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("web client is not built: " + err.Error()))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}
