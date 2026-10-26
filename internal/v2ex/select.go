package v2ex

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// findAttrPrefix returns the first descendant whose attr starts with prefix.
func findAttrPrefix(root *goquery.Selection, selector, attr, prefix string) *goquery.Selection {
	var out *goquery.Selection
	root.Find(selector).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		if v, ok := s.Attr(attr); ok && strings.HasPrefix(v, prefix) {
			out = s
			return false
		}
		return true
	})
	return out
}

// findAttrEqual returns the first element with attr == value.
func findAttrEqual(root *goquery.Selection, selector, attr, value string) *goquery.Selection {
	var out *goquery.Selection
	root.Find(selector).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		if v, ok := s.Attr(attr); ok && v == value {
			out = s
			return false
		}
		return true
	})
	return out
}

// firstNonEmpty returns the first non-empty element of a selection.
func firstNonEmpty(sel *goquery.Selection) *goquery.Selection {
	var out *goquery.Selection
	sel.EachWithBreak(func(_ int, s *goquery.Selection) bool {
		if strings.TrimSpace(s.Text()) != "" {
			out = s
			return false
		}
		return true
	})
	return out
}
