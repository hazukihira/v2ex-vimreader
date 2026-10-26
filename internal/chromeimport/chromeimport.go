// Package chromeimport reads cookies from a locally running Chrome through the
// DevTools protocol. It never touches Chrome profile files directly, so it
// works with Chrome's app-bound cookie encryption and open profiles.
package chromeimport

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var (
	// ErrNotConfigured means Chrome remote debugging is not reachable.
	ErrNotConfigured = errors.New("无法连接 Chrome：请确认 Chrome 已开启远程调试（chrome://inspect/#remote-debugging），并在 Chrome 的允许提示中点击 Allow")
	// ErrNoCookies means Chrome has no v2ex.com session cookie.
	ErrNoCookies = errors.New("Chrome 里没有 v2ex.com 的 Cookie，请先在 Chrome 登录 v2ex.com")
)

type cdpCookie struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain"`
}

func userDataDir() string {
	if dir := strings.TrimSpace(os.Getenv("CHROME_USER_DATA_DIR")); dir != "" {
		return dir
	}
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "User Data")
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Google", "Chrome")
	default:
		return filepath.Join(os.Getenv("HOME"), ".config", "google-chrome")
	}
}

// CookieHeader connects to Chrome and returns a "name=value; ..." header for
// v2ex.com. The context deadline covers waiting for the user to approve the
// DevTools connection.
func CookieHeader(ctx context.Context) (string, error) {
	portFile := filepath.Join(userDataDir(), "DevToolsActivePort")
	raw, err := os.ReadFile(portFile)
	if err != nil {
		return "", fmt.Errorf("%w（读取 %s 失败：%v）", ErrNotConfigured, portFile, err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) < 2 {
		return "", fmt.Errorf("%w（DevToolsActivePort 内容不完整）", ErrNotConfigured)
	}
	port := strings.TrimSpace(lines[0])
	wsPath := strings.TrimSpace(lines[1])
	if port == "" || wsPath == "" {
		return "", fmt.Errorf("%w（DevToolsActivePort 内容不完整）", ErrNotConfigured)
	}
	dialer := websocket.Dialer{HandshakeTimeout: 15 * time.Second}
	conn, resp, err := dialer.DialContext(ctx, "ws://127.0.0.1:"+port+wsPath, nil)
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return "", fmt.Errorf("%w（%v）", ErrNotConfigured, err)
	}
	defer conn.Close()
	deadline := time.Now().Add(2 * time.Minute)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	_ = conn.SetReadDeadline(deadline)
	if err := conn.WriteJSON(map[string]any{
		"id":     1,
		"method": "Storage.getCookies",
		"params": map[string]any{},
	}); err != nil {
		return "", fmt.Errorf("%w（发送 DevTools 请求失败：%v）", ErrNotConfigured, err)
	}
	var message struct {
		ID     int `json:"id"`
		Result struct {
			Cookies []cdpCookie `json:"cookies"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	for {
		if err := conn.ReadJSON(&message); err != nil {
			return "", fmt.Errorf("%w（读取 DevTools 响应失败：%v）", ErrNotConfigured, err)
		}
		if message.ID == 1 {
			break
		}
	}
	if message.Error != nil {
		return "", fmt.Errorf("%w（DevTools 错误：%s）", ErrNotConfigured, message.Error.Message)
	}
	header := cookieHeader(message.Result.Cookies)
	if header == "" {
		return "", ErrNoCookies
	}
	return header, nil
}

func cookieHeader(cookies []cdpCookie) string {
	seen := make(map[string]bool, len(cookies))
	parts := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		if !matchesV2EX(cookie.Domain) || cookie.Name == "" {
			continue
		}
		if seen[cookie.Name] {
			continue
		}
		seen[cookie.Name] = true
		parts = append(parts, cookie.Name+"="+cookie.Value)
	}
	return strings.Join(parts, "; ")
}

func matchesV2EX(domain string) bool {
	d := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(domain), "."))
	return d == "v2ex.com" || strings.HasSuffix(d, ".v2ex.com")
}
