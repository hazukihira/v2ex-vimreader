package service

import (
	"context"
	"strings"
)

// isRestrictedNode reports whether a node is login-only. The configured list is
// authoritative, and runtime probes can additionally mark nodes through the
// node directory (for example when a page returns a login wall).
func (s *Service) isRestrictedNode(ctx context.Context, name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return false
	}
	if s.loggedIn() {
		return false
	}
	if _, ok := s.restricted[name]; ok {
		return true
	}
	if n, err := s.st.GetNode(ctx, name); err == nil && n.Restricted {
		return true
	}
	return false
}

// RestrictedNodeNames returns the configured restricted node names.
func (s *Service) RestrictedNodeNames() []string {
	out := make([]string, 0, len(s.restricted))
	for name := range s.restricted {
		out = append(out, name)
	}
	return out
}
