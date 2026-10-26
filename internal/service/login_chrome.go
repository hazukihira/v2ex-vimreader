package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hazukihira/v2ex-vimreader/internal/chromeimport"
)

// ImportChromeCookie reads the current v2ex.com cookies from a locally running
// Chrome (DevTools protocol) and stores them as the reader session.
func (s *Service) ImportChromeCookie(ctx context.Context) (*LoginStatus, error) {
	importCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	header, err := chromeimport.CookieHeader(importCtx)
	if err != nil {
		if errors.Is(err, chromeimport.ErrNoCookies) {
			return nil, fmt.Errorf("%w：%v", ErrChromeUnavailable, chromeimport.ErrNoCookies)
		}
		return nil, fmt.Errorf("%w：%v", ErrChromeUnavailable, err)
	}
	return s.SetLoginCookie(ctx, header)
}
