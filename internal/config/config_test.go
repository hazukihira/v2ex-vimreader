package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != "127.0.0.1:7788" {
		t.Fatalf("unexpected listen: %s", cfg.Listen)
	}
	if cfg.Fetch.RequestsPerSecond <= 0 || cfg.Fetch.MaxRetries < 1 {
		t.Fatalf("unexpected fetcher defaults: %+v", cfg.Fetch)
	}
	if len(cfg.Sync.FollowedNodes) == 0 {
		t.Fatalf("followed nodes are empty")
	}
	if cfg.Location() == nil {
		t.Fatalf("nil location")
	}
	if cfg.Login.Cookie != "" {
		t.Fatalf("unexpected default login cookie: %q", cfg.Login.Cookie)
	}
}

func TestFileOverrideAndEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := "listen = \"127.0.0.1:9000\"\n[ai]\nenabled = true\nmodel = \"test-model\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VIMREADER_LISTEN", "127.0.0.1:9100")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != "127.0.0.1:9100" {
		t.Fatalf("env override failed: %s", cfg.Listen)
	}
	if !cfg.AI.Enabled || cfg.AI.Model != "test-model" {
		t.Fatalf("file override failed: %+v", cfg.AI)
	}
	if cfg.Fetch.TimeoutSeconds == 0 {
		t.Fatalf("defaults were not preserved")
	}
}

func TestResolveDataDir(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	cfg.DataDir = filepath.Join(t.TempDir(), "v2ex-vimreader")
	dir, err := cfg.ResolveDataDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir == "" {
		t.Fatalf("empty data dir")
	}
}
