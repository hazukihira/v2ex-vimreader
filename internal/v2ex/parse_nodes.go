package v2ex

import (
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/hazukihira/v2ex-vimreader/internal/model"
)

// ParsePlanes parses /planes (the node directory page).
func ParsePlanes(doc *goquery.Document) []model.Node {
	seen := make(map[string]struct{}, 1500)
	nodes := make([]model.Node, 0, 1400)
	doc.Find("a").Each(func(_ int, a *goquery.Selection) {
		href, ok := a.Attr("href")
		if !ok || !strings.HasPrefix(href, "/go/") {
			return
		}
		name := strings.TrimSpace(strings.Trim(strings.TrimPrefix(href, "/go/"), "/"))
		title := strings.TrimSpace(a.Text())
		if name == "" || title == "" {
			return
		}
		if _, dup := seen[name]; dup {
			return
		}
		seen[name] = struct{}{}
		nodes = append(nodes, model.Node{Name: name, Title: title})
	})
	return nodes
}
