package chromeimport

import "testing"

func TestCookieHeaderFiltersV2EXAndDeduplicates(t *testing.T) {
	cookies := []cdpCookie{
		{Name: "A2", Value: "one", Domain: ".v2ex.com"},
		{Name: "PB3_SESSION", Value: "two", Domain: "v2ex.com"},
		{Name: "A2", Value: "dup", Domain: "www.v2ex.com"},
		{Name: "OTHER", Value: "x", Domain: ".example.com"},
		{Name: "V2EX_LANG", Value: "zhcn", Domain: "v2ex.com"},
	}
	header := cookieHeader(cookies)
	want := "A2=one; PB3_SESSION=two; V2EX_LANG=zhcn"
	if header != want {
		t.Fatalf("cookie header = %q, want %q", header, want)
	}
}

func TestMatchesV2EX(t *testing.T) {
	for _, domain := range []string{"v2ex.com", ".v2ex.com", "www.v2ex.com", "V2EX.COM"} {
		if !matchesV2EX(domain) {
			t.Errorf("%q should match", domain)
		}
	}
	for _, domain := range []string{"example.com", "v2ex.com.evil.test", "notv2ex.com"} {
		if matchesV2EX(domain) {
			t.Errorf("%q should not match", domain)
		}
	}
}
