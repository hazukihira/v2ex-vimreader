package service

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/hazukihira/v2ex-vimreader/internal/config"
)

func TestLoginStateLoadsFromDatabase(t *testing.T) {
	ctx := context.Background()
	cfg := config.Default()
	st := mustStore(t)
	now := time.Now().Unix()
	for key, value := range map[string]string{
		loginCookieMeta:  "PB3_SESSION=test-cookie",
		loginValidMeta:   "1",
		loginUserMeta:    "testuser",
		loginCheckedMeta: strconv.FormatInt(now, 10),
	} {
		if err := st.SetMeta(ctx, key, value); err != nil {
			t.Fatal(err)
		}
	}
	svc := New(&cfg, st, discardLogger())
	status := svc.LoginStatus()
	if !status.HasCookie || !status.LoggedIn || status.User != "testuser" {
		t.Fatalf("unexpected loaded status: %+v", status)
	}
	if svc.vc.Cookie() != "PB3_SESSION=test-cookie" {
		t.Fatalf("client cookie not restored")
	}
	if svc.isRestrictedNode(ctx, "nsfw") {
		t.Fatalf("logged-in service should not treat nsfw as restricted")
	}
}

func TestConfigCookieAndClear(t *testing.T) {
	ctx := context.Background()
	cfg := config.Default()
	cfg.Login.Cookie = "PB3_SESSION=config-cookie"
	st := mustStore(t)
	svc := New(&cfg, st, discardLogger())
	if !svc.LoginStatus().HasCookie {
		t.Fatalf("config cookie was not loaded")
	}
	if !svc.isRestrictedNode(ctx, "nsfw") {
		t.Fatalf("unverified cookie must not unlock restricted nodes")
	}
	status, err := svc.ClearLoginCookie(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.HasCookie || status.LoggedIn {
		t.Fatalf("login was not cleared: %+v", status)
	}
	if svc.vc.Cookie() != "" {
		t.Fatalf("client cookie was not cleared")
	}
}

func TestImportChromeCookieWithoutChrome(t *testing.T) {
	t.Setenv("CHROME_USER_DATA_DIR", t.TempDir())
	cfg := config.Default()
	svc := New(&cfg, mustStore(t), discardLogger())
	_, err := svc.ImportChromeCookie(context.Background())
	if !errors.Is(err, ErrChromeUnavailable) {
		t.Fatalf("want ErrChromeUnavailable, got %v", err)
	}
}
