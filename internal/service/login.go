package service

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/hazukihira/v2ex-vimreader/internal/v2ex"
)

const (
	loginCookieMeta  = "login_cookie"
	loginValidMeta   = "login_valid"
	loginUserMeta    = "login_user"
	loginCheckedMeta = "login_checked_at"
)

type loginResult struct {
	user string
	ok   bool
}

// loadLogin restores the login state from the database or the config file.
// A database state (even a cleared one) takes precedence over the config file
// once the user has interacted with the login UI.
func (s *Service) loadLogin() {
	ctx := context.Background()
	cookie := strings.TrimSpace(s.cfg.Login.Cookie)
	checkedAt := s.metaInt(ctx, loginCheckedMeta)
	if checkedAt > 0 {
		if v, err := s.st.GetMeta(ctx, loginCookieMeta); err == nil {
			cookie = strings.TrimSpace(v)
		}
	}
	user := ""
	if v, err := s.st.GetMeta(ctx, loginUserMeta); err == nil {
		user = v
	}
	valid := false
	if v, err := s.st.GetMeta(ctx, loginValidMeta); err == nil && v == "1" {
		valid = true
	}
	s.loginMu.Lock()
	s.loginCookie = cookie
	s.loginUser = user
	s.loginLoggedIn = cookie != "" && valid
	s.loginCheckedAt = checkedAt
	s.loginChecked = checkedAt > 0
	s.loginMu.Unlock()
	if cookie != "" {
		s.vc.SetCookie(cookie)
	}
}

// loggedIn reports whether a validated session cookie is active.
func (s *Service) loggedIn() bool {
	s.loginMu.RLock()
	defer s.loginMu.RUnlock()
	return s.loginLoggedIn && s.loginCookie != ""
}

// LoginStatus returns the current state without exposing the cookie.
func (s *Service) LoginStatus() LoginStatus {
	s.loginMu.RLock()
	defer s.loginMu.RUnlock()
	status := LoginStatus{
		HasCookie: s.loginCookie != "",
		LoggedIn:  s.loginLoggedIn,
		CheckedAt: s.loginCheckedAt,
	}
	if status.LoggedIn {
		status.User = s.loginUser
	}
	return status
}

func (s *Service) loginSnapshot() (string, bool, string, int64) {
	s.loginMu.RLock()
	defer s.loginMu.RUnlock()
	return s.loginCookie, s.loginLoggedIn, s.loginUser, s.loginCheckedAt
}

func (s *Service) restoreLogin(cookie string, loggedIn bool, user string, checkedAt int64) {
	s.loginMu.Lock()
	s.loginCookie = cookie
	s.loginLoggedIn = loggedIn
	s.loginUser = user
	s.loginCheckedAt = checkedAt
	s.loginChecked = checkedAt > 0
	s.loginMu.Unlock()
	s.vc.SetCookie(cookie)
}

// verifyLogin fetches the home page with the current cookie and reads the
// signed-in member from the top navigation. A missing link means the cookie is
// invalid or expired. It never returns partial results on network errors.
func (s *Service) verifyLogin(ctx context.Context) (string, bool, error) {
	value, err := s.flights.Do("login-verify", func() (any, error) {
		if strings.TrimSpace(s.vc.Cookie()) == "" {
			return loginResult{}, nil
		}
		resp, err := s.vc.Get(ctx, v2ex.SiteURL("/"))
		if err != nil {
			return nil, err
		}
		if err := resp.Err(); err != nil {
			return nil, err
		}
		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(resp.Body))
		if err != nil {
			return nil, err
		}
		user := ""
		doc.Find("#Top a").EachWithBreak(func(_ int, sel *goquery.Selection) bool {
			href, ok := sel.Attr("href")
			if !ok || !strings.HasPrefix(href, "/member/") {
				return true
			}
			user = strings.TrimSpace(sel.Text())
			return false
		})
		return loginResult{user: user, ok: user != "" && !resp.IsLoginWall()}, nil
	})
	if err != nil {
		return "", false, err
	}
	result := value.(loginResult)
	return result.user, result.ok, nil
}

// SetLoginCookie validates then persists a V2EX cookie. Passing an empty string
// clears the current login.
func (s *Service) SetLoginCookie(ctx context.Context, cookie string) (*LoginStatus, error) {
	cookie = strings.TrimSpace(cookie)
	if cookie == "" {
		return s.ClearLoginCookie(ctx)
	}
	prevCookie, prevLoggedIn, prevUser, prevCheckedAt := s.loginSnapshot()
	s.loginMu.Lock()
	s.loginCookie = cookie
	s.loginLoggedIn = false
	s.loginMu.Unlock()
	s.vc.SetCookie(cookie)
	user, ok, err := s.verifyLogin(ctx)
	if err != nil {
		s.restoreLogin(prevCookie, prevLoggedIn, prevUser, prevCheckedAt)
		return nil, err
	}
	if !ok {
		s.restoreLogin(prevCookie, prevLoggedIn, prevUser, prevCheckedAt)
		return nil, ErrLoginFailed
	}
	now := time.Now().Unix()
	s.loginMu.Lock()
	s.loginLoggedIn = true
	s.loginUser = user
	s.loginChecked = true
	s.loginCheckedAt = now
	s.loginMu.Unlock()
	s.persistLogin(ctx, cookie, true, user, now)
	status := s.LoginStatus()
	return &status, nil
}

// ClearLoginCookie forgets the local V2EX session.
func (s *Service) ClearLoginCookie(ctx context.Context) (*LoginStatus, error) {
	now := time.Now().Unix()
	s.restoreLogin("", false, "", now)
	s.persistLogin(ctx, "", false, "", now)
	status := s.LoginStatus()
	return &status, nil
}

func (s *Service) persistLogin(ctx context.Context, cookie string, valid bool, user string, checkedAt int64) {
	set := func(key, value string) {
		if err := s.st.SetMeta(ctx, key, value); err != nil {
			s.log.Warn("save login state failed", "key", key, "err", err)
		}
	}
	set(loginCookieMeta, cookie)
	if valid {
		set(loginValidMeta, "1")
	} else {
		set(loginValidMeta, "0")
	}
	set(loginUserMeta, user)
	set(loginCheckedMeta, strconv.FormatInt(checkedAt, 10))
}

// loginLoop re-validates the session periodically so an expired cookie is
// reflected in the status bar instead of failing every restricted request.
func (s *Service) loginLoop(ctx context.Context) {
	if strings.TrimSpace(s.vc.Cookie()) == "" {
		return
	}
	s.checkLogin(ctx)
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkLogin(ctx)
		}
	}
}

func (s *Service) checkLogin(ctx context.Context) {
	user, ok, err := s.verifyLogin(ctx)
	if err != nil {
		s.log.Debug("login check failed", "err", err)
		return
	}
	now := time.Now().Unix()
	s.loginMu.Lock()
	s.loginLoggedIn = ok
	s.loginChecked = true
	s.loginCheckedAt = now
	if ok {
		s.loginUser = user
	} else {
		s.loginUser = ""
	}
	cookie := s.loginCookie
	s.loginMu.Unlock()
	s.persistLogin(ctx, cookie, ok, user, now)
	if !ok {
		s.log.Warn("V2EX login cookie is no longer valid")
	} else {
		s.log.Info("V2EX login active", "user", user)
	}
}
