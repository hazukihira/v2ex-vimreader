package v2ex

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hazukihira/v2ex-vimreader/internal/config"
)

const baseURL = "https://www.v2ex.com"

// Client is a rate limited HTTP client for v2ex.com.
type Client struct {
	hc        *http.Client
	ua        string
	limiter   *Limiter
	retries   int
	maxBody   int64
	log       *slog.Logger
	remaining atomic.Int32
	cookieMu  sync.RWMutex
	cookie    string
}

// Options are per-request headers.
type Options struct {
	ETag         string
	LastModified string
	Accept       string
}

// Option mutates Options.
type Option func(*Options)

// WithETag sets If-None-Match.
func WithETag(etag string) Option { return func(o *Options) { o.ETag = etag } }

// WithLastModified sets If-Modified-Since.
func WithLastModified(lm string) Option { return func(o *Options) { o.LastModified = lm } }

// WithAccept overrides the Accept header.
func WithAccept(accept string) Option { return func(o *Options) { o.Accept = accept } }

// Response is a completed HTTP response.
type Response struct {
	StatusCode  int
	Body        []byte
	Header      http.Header
	FinalURL    *url.URL
	NotModified bool
	Attempts    int
}

// HTTPError describes a non-2xx response.
type HTTPError struct {
	StatusCode int
	URL        string
	Body       string
}

func (e *HTTPError) Error() string {
	msg := strings.TrimSpace(e.Body)
	if len(msg) > 200 {
		msg = msg[:200]
	}
	return fmt.Sprintf("v2ex: HTTP %d for %s: %s", e.StatusCode, e.URL, msg)
}

// IsLoginWall reports whether the final URL is the sign-in page.
func (r *Response) IsLoginWall() bool {
	if r == nil || r.FinalURL == nil {
		return false
	}
	return strings.HasPrefix(r.FinalURL.Path, "/signin") || strings.HasPrefix(r.FinalURL.Path, "/restricted")
}

// Err returns an HTTPError for non-2xx responses.
func (r *Response) Err() error {
	if r == nil {
		return fmt.Errorf("v2ex: empty response")
	}
	if r.StatusCode >= 200 && r.StatusCode < 300 {
		return nil
	}
	return &HTTPError{StatusCode: r.StatusCode, URL: r.FinalURL.String(), Body: string(r.Body)}
}

// NewClient creates the shared HTTP client.
func NewClient(cfg config.FetchConfig, log *slog.Logger) *Client {
	dialer := &net.Dialer{Timeout: 20 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          16,
		MaxIdleConnsPerHost:   8,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   30 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 2 * time.Second,
	}
	return &Client{
		hc: &http.Client{
			Transport: transport,
			Timeout:   time.Duration(cfg.TimeoutSeconds) * time.Second,
		},
		ua:      cfg.UserAgent,
		limiter: NewLimiter(cfg.RequestsPerSecond, cfg.Burst),
		retries: cfg.MaxRetries,
		maxBody: 8 << 20,
		log:     log,
	}
}

// Limiter exposes rate limiter statistics.
func (c *Client) Limiter() *Limiter { return c.limiter }

// Remaining returns the last X-Rate-Limit-Remaining value seen (0 when unknown).
func (c *Client) Remaining() int { return int(c.remaining.Load()) }

// SetCookie sets the Cookie header sent with every request. Empty disables it.
func (c *Client) SetCookie(cookie string) {
	c.cookieMu.Lock()
	c.cookie = strings.TrimSpace(cookie)
	c.cookieMu.Unlock()
}

// Cookie returns the current Cookie header value.
func (c *Client) Cookie() string {
	c.cookieMu.RLock()
	defer c.cookieMu.RUnlock()
	return c.cookie
}

// Get fetches rawURL, retrying network errors and 5xx/429 responses.
func (c *Client) Get(ctx context.Context, rawURL string, opts ...Option) (*Response, error) {
	o := Options{Accept: "text/html,application/xhtml+xml,application/atom+xml,application/json;q=0.9,*/*;q=0.8"}
	for _, fn := range opts {
		fn(&o)
	}
	attempts := c.retries + 1
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, err
		}
		resp, err := c.do(ctx, rawURL, o)
		if err != nil {
			lastErr = err
			if attempt < attempts {
				c.backoff(ctx, attempt)
				continue
			}
			return nil, err
		}
		resp.Attempts = attempt
		if resp.StatusCode == http.StatusNotModified {
			resp.NotModified = true
			return resp, nil
		}
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			lastErr = resp.Err()
			if attempt < attempts {
				c.log.Debug("retrying v2ex request", "status", resp.StatusCode, "url", rawURL)
				c.backoff(ctx, attempt)
				continue
			}
			return resp, lastErr
		}
		return resp, nil
	}
	return nil, lastErr
}

func (c *Client) backoff(ctx context.Context, attempt int) {
	d := time.Duration(1<<uint(attempt-1)) * 1200 * time.Millisecond
	if d > 6*time.Second {
		d = 6 * time.Second
	}
	d += time.Duration(rand.Intn(500)) * time.Millisecond
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

func (c *Client) do(ctx context.Context, rawURL string, o Options) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.ua)
	req.Header.Set("Accept", o.Accept)
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.6")
	if cookie := c.Cookie(); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	if o.ETag != "" {
		req.Header.Set("If-None-Match", o.ETag)
	}
	if o.LastModified != "" {
		req.Header.Set("If-Modified-Since", o.LastModified)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxBody+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > c.maxBody {
		return nil, fmt.Errorf("v2ex: response larger than %d bytes", c.maxBody)
	}
	if v := resp.Header.Get("x-rate-limit-remaining"); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			c.remaining.Store(int32(n))
		}
	}
	return &Response{StatusCode: resp.StatusCode, Body: body, Header: resp.Header, FinalURL: resp.Request.URL}, nil
}
