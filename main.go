package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/hazukihira/v2ex-vimreader/internal/api"
	"github.com/hazukihira/v2ex-vimreader/internal/config"
	"github.com/hazukihira/v2ex-vimreader/internal/service"
	"github.com/hazukihira/v2ex-vimreader/internal/store"
)

//go:embed all:web
var webAssets embed.FS

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "v2ex-vimreader:", err)
		os.Exit(1)
	}
}

func run() error {
	var configPath, listen, dataDir, webDir, cookieFlag string
	var noOpen, debug, showVersion, logoutFlag bool
	flag.StringVar(&configPath, "config", "", "path to config.toml (default: ./config.toml or next to the executable)")
	flag.StringVar(&listen, "listen", "", "listen address, for example 127.0.0.1:7788")
	flag.StringVar(&dataDir, "data", "", "data directory for the SQLite database")
	flag.StringVar(&webDir, "web-dir", "", "serve the web client from this directory instead of the embedded copy (development)")
	flag.BoolVar(&noOpen, "no-open", false, "do not open the browser on startup")
	flag.BoolVar(&debug, "debug", false, "enable debug logging")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.StringVar(&cookieFlag, "cookie", "", "V2EX cookie header to validate and store locally, then exit after startup")
	flag.BoolVar(&logoutFlag, "logout", false, "clear the stored V2EX login cookie")
	flag.Parse()

	if showVersion {
		fmt.Println("v2ex-vimreader " + config.Version)
		return nil
	}

	if configPath == "" {
		configPath = findConfig()
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	// config.toml 里的相对 data_dir 以配置文件所在目录为基准，
	// 这样从 bin/ 直接双击 exe 也能找到项目根目录下的 data/（登录 Cookie 才会持续存在）。
	if dataDir == "" && cfg.DataDir != "" && !filepath.IsAbs(cfg.DataDir) && configPath != "" {
		cfg.DataDir = filepath.Join(filepath.Dir(configPath), cfg.DataDir)
	}
	if listen != "" {
		cfg.Listen = listen
	}
	if dataDir != "" {
		cfg.DataDir = dataDir
	}

	level := slog.LevelInfo
	if debug || cfg.LogLevel == "debug" {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	dir, err := cfg.ResolveDataDir()
	if err != nil {
		return err
	}
	dbPath := filepath.Join(dir, "v2ex-vimreader.db")
	st, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer st.Close()

	svc := service.New(cfg, st, logger)
	if logoutFlag {
		if _, err := svc.ClearLoginCookie(context.Background()); err != nil {
			logger.Warn("logout failed", "err", err)
		} else {
			fmt.Println("已清除本地 V2EX 登录 Cookie")
		}
	}
	if cookieFlag != "" {
		status, err := svc.SetLoginCookie(context.Background(), cookieFlag)
		if err != nil {
			fmt.Fprintln(os.Stderr, "登录校验失败:", err)
		} else {
			fmt.Println("已登录 V2EX:", status.User)
		}
	}
	svc.StartBackground()
	defer svc.Stop()

	var webRoot fs.FS
	if webDir != "" {
		webRoot = os.DirFS(webDir)
	} else {
		sub, err := fs.Sub(webAssets, "web")
		if err != nil {
			return err
		}
		webRoot = sub
	}
	serverOptions := []api.Option{}
	if webDir != "" {
		serverOptions = append(serverOptions, api.WithDevMode())
	}
	httpServer := &http.Server{
		Addr:              cfg.Listen,
		Handler:           api.NewServer(svc, webRoot, logger, serverOptions...),
		ReadHeaderTimeout: 10 * time.Second,
	}
	listener, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		return fmt.Errorf("listen on %s: %w (is another copy already running?)", cfg.Listen, err)
	}
	url := "http://" + cfg.Listen
	logger.Info("v2ex-vimreader started", "url", url, "data_dir", dir, "version", config.Version)
	fmt.Println("v2ex-vimreader " + config.Version)
	fmt.Println("  URL:  " + url)
	fmt.Println("  data: " + dir)
	fmt.Println("  quit: Ctrl+C")

	if !noOpen {
		go openBrowser(url)
	}

	serveErr := make(chan error, 1)
	go func() {
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			serveErr <- err
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	select {
	case err := <-serveErr:
		return err
	case <-sig:
	}
	logger.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
	return nil
}

func findConfig() string {
	candidates := []string{"config.toml"}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "config.toml"),
			filepath.Join(exeDir, "..", "config.toml"),
		)
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if cmd != nil {
		_ = cmd.Start()
	}
}
