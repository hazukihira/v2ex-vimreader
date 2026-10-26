// Package config loads the TOML configuration with sane defaults.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "time/tzdata" // keep time zone data available on Windows builds

	"github.com/BurntSushi/toml"
)

// Config is the full application configuration.
type Config struct {
	Listen   string `toml:"listen"`
	DataDir  string `toml:"data_dir"`
	LogLevel string `toml:"log_level"`
	Timezone string `toml:"timezone"`

	Fetch FetchConfig `toml:"fetch"`
	Sync  SyncConfig  `toml:"sync"`
	AI    AIConfig    `toml:"ai"`
	Login LoginConfig `toml:"login"`
}

// LoginConfig holds the optional V2EX login cookie. The cookie is only used
// locally to read pages that require a signed-in session; it is never exposed
// through the HTTP API. Prefer configuring it through the web UI (stored in
// the local SQLite database) or the -cookie flag.
type LoginConfig struct {
	Cookie string `toml:"cookie"`
}

// FetchConfig controls the HTTP client, caching and polling intervals.
type FetchConfig struct {
	UserAgent           string  `toml:"user_agent"`
	RequestsPerSecond   float64 `toml:"requests_per_second"`
	Burst               int     `toml:"burst"`
	TimeoutSeconds      int     `toml:"timeout_seconds"`
	MaxRetries          int     `toml:"max_retries"`
	ListCacheSeconds    int     `toml:"list_cache_seconds"`
	TopicCacheSeconds   int     `toml:"topic_cache_seconds"`
	SiteFeedPollSeconds int     `toml:"site_feed_poll_seconds"`
	NodeFeedPollSeconds int     `toml:"node_feed_poll_seconds"`
}

// SyncConfig controls the background incremental sync.
type SyncConfig struct {
	Enabled       bool     `toml:"enabled"`
	SiteFeed      bool     `toml:"site_feed"`
	FollowedNodes []string `toml:"followed_nodes"`
	// RestrictedNodes are login-only nodes. They are marked in the node
	// directory and never ingested via feeds, even though some feeds are
	// readable anonymously.
	RestrictedNodes []string `toml:"restricted_nodes"`
}

// AIConfig configures an OpenAI-compatible chat completion endpoint.
type AIConfig struct {
	Enabled        bool              `toml:"enabled"`
	BaseURL        string            `toml:"base_url"`
	Model          string            `toml:"model"`
	APIKey         string            `toml:"api_key"`
	APIKeyEnv      string            `toml:"api_key_env"`
	TimeoutSeconds int               `toml:"timeout_seconds"`
	MaxInputChars  int               `toml:"max_input_chars"`
	Prompt         string            `toml:"prompt"`
	Headers        map[string]string `toml:"headers"`
}

// Version is the application version, overridden at build time when needed.
var Version = "0.1.0"

// Default returns the built-in defaults.
func Default() Config {
	return Config{
		Listen:   "127.0.0.1:7788",
		DataDir:  "",
		LogLevel: "info",
		Timezone: "Asia/Shanghai",
		Fetch: FetchConfig{
			UserAgent:           "v2ex-vimreader/" + Version + " (+https://github.com/hazukihira/v2ex-vimreader)",
			RequestsPerSecond:   1.0,
			Burst:               2,
			TimeoutSeconds:      25,
			MaxRetries:          3,
			ListCacheSeconds:    120,
			TopicCacheSeconds:   300,
			SiteFeedPollSeconds: 600,
			NodeFeedPollSeconds: 1800,
		},
		Sync: SyncConfig{
			Enabled:  true,
			SiteFeed: true,
			FollowedNodes: []string{
				"tech", "qna", "jobs", "create", "ideas", "apple", "play",
				"travel", "life", "movie", "music", "reading", "career",
				"programmer", "android", "ios", "python", "go",
				"linux", "windows", "hardware", "design", "share", "deals",
				"invest", "sidehustle", "product", "remote", "games", "openai",
			},
			RestrictedNodes: []string{
				"nsfw", "sandbox", "chamber", "cheap", "flamewar", "flood", "monthly", "survey",
			},
		},
		Login: LoginConfig{},
		AI: AIConfig{
			Enabled:        false,
			BaseURL:        "https://api.deepseek.com/v1",
			Model:          "deepseek-chat",
			APIKeyEnv:      "VIMREADER_AI_KEY",
			TimeoutSeconds: 120,
			MaxInputChars:  60000,
		},
	}
}

// Load reads path (optional) on top of the defaults, then applies env overrides.
func Load(path string) (*Config, error) {
	cfg := Default()
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if _, err := toml.DecodeFile(path, &cfg); err != nil {
				return nil, fmt.Errorf("decode config %s: %w", path, err)
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat config %s: %w", path, err)
		}
	}
	applyEnv(&cfg)
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func applyEnv(cfg *Config) {
	if v := strings.TrimSpace(os.Getenv("VIMREADER_LISTEN")); v != "" {
		cfg.Listen = v
	}
	if v := strings.TrimSpace(os.Getenv("VIMREADER_DATA_DIR")); v != "" {
		cfg.DataDir = v
	}
	if v := strings.TrimSpace(os.Getenv("VIMREADER_LOG_LEVEL")); v != "" {
		cfg.LogLevel = v
	}
}

func (c *Config) validate() error {
	if c.Listen == "" {
		return fmt.Errorf("listen address is empty")
	}
	if c.Fetch.RequestsPerSecond <= 0 {
		c.Fetch.RequestsPerSecond = 1
	}
	if c.Fetch.RequestsPerSecond > 20 {
		c.Fetch.RequestsPerSecond = 20
	}
	if c.Fetch.Burst < 1 {
		c.Fetch.Burst = 1
	}
	if c.Fetch.TimeoutSeconds <= 0 {
		c.Fetch.TimeoutSeconds = 20
	}
	if c.Fetch.MaxRetries < 0 {
		c.Fetch.MaxRetries = 0
	}
	if c.Fetch.MaxRetries > 5 {
		c.Fetch.MaxRetries = 5
	}
	if c.Fetch.ListCacheSeconds < 0 {
		c.Fetch.ListCacheSeconds = 0
	}
	if c.Fetch.TopicCacheSeconds < 0 {
		c.Fetch.TopicCacheSeconds = 0
	}
	if c.AI.TimeoutSeconds <= 0 {
		c.AI.TimeoutSeconds = 120
	}
	if c.AI.MaxInputChars <= 0 {
		c.AI.MaxInputChars = 60000
	}
	if c.Timezone == "" {
		c.Timezone = "Asia/Shanghai"
	}
	if _, err := time.LoadLocation(c.Timezone); err != nil {
		return fmt.Errorf("invalid timezone %q: %w", c.Timezone, err)
	}
	return nil
}

// Location returns the configured time zone.
func (c *Config) Location() *time.Location {
	loc, err := time.LoadLocation(c.Timezone)
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

// ResolveDataDir returns the absolute directory used for the database.
func (c *Config) ResolveDataDir() (string, error) {
	dir := c.DataDir
	if dir == "" {
		switch runtime.GOOS {
		case "windows":
			dir = os.Getenv("LOCALAPPDATA")
			if dir == "" {
				dir = os.Getenv("APPDATA")
			}
			if dir != "" {
				dir = filepath.Join(dir, "v2ex-vimreader")
			}
		case "darwin":
			home, _ := os.UserHomeDir()
			dir = filepath.Join(home, "Library", "Application Support", "v2ex-vimreader")
		default:
			if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
				dir = filepath.Join(xdg, "v2ex-vimreader")
			} else {
				home, _ := os.UserHomeDir()
				dir = filepath.Join(home, ".local", "share", "v2ex-vimreader")
			}
		}
	}
	if dir == "" {
		return "", fmt.Errorf("cannot determine data dir, please set data_dir in config")
	}
	if strings.HasPrefix(dir, "~") {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, strings.TrimPrefix(dir, "~"))
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", fmt.Errorf("create data dir: %w", err)
	}
	return abs, nil
}

// AIAPIKey resolves the API key from config or the configured env variable.
func (c *Config) AIAPIKey() string {
	if c.AI.APIKeyEnv != "" {
		if v := strings.TrimSpace(os.Getenv(c.AI.APIKeyEnv)); v != "" {
			return v
		}
	}
	return strings.TrimSpace(c.AI.APIKey)
}

// Durations are helpers that turn the integer seconds into time.Duration.
func (f FetchConfig) ListCacheTTL() time.Duration {
	return time.Duration(f.ListCacheSeconds) * time.Second
}
func (f FetchConfig) TopicCacheTTL() time.Duration {
	return time.Duration(f.TopicCacheSeconds) * time.Second
}
