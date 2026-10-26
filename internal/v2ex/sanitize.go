package v2ex

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/microcosm-cc/bluemonday"
)

var allowedEmbedHosts = map[string]bool{
	"youtube.com":              true,
	"www.youtube.com":          true,
	"youtube-nocookie.com":     true,
	"www.youtube-nocookie.com": true,
	"player.bilibili.com":      true,
}

var classAllowList = regexp.MustCompile(`^(embedded_video_wrapper|embedded_video|topic_content|reply_content|markdown_body)$`)

// Sanitizer cleans untrusted V2EX HTML before it is stored and served.
type Sanitizer struct {
	p *bluemonday.Policy
}

// NewSanitizer builds the allow-list policy.
func NewSanitizer() *Sanitizer {
	p := bluemonday.NewPolicy()
	p.AllowStandardURLs()
	p.AllowElements(
		"p", "br", "strong", "b", "em", "i", "u", "s", "del", "ins",
		"code", "pre", "blockquote", "ul", "ol", "li", "hr",
		"h1", "h2", "h3", "h4", "h5", "h6",
		"span", "div", "table", "thead", "tbody", "tr", "td", "th",
		"sub", "sup", "img", "iframe", "a",
	)
	p.AllowAttrs("href", "title").OnElements("a")
	p.AllowAttrs("src", "alt", "title", "width", "height", "loading", "referrerpolicy").OnElements("img")
	p.AllowAttrs("src", "title", "width", "height", "allowfullscreen", "frameborder", "loading").OnElements("iframe")
	p.AllowAttrs("class").Matching(classAllowList).OnElements("div", "span", "iframe")
	p.RequireNoFollowOnLinks(true)
	p.AddTargetBlankToFullyQualifiedLinks(true)
	p.RequireNoReferrerOnLinks(true)
	return &Sanitizer{p: p}
}

// Prepare normalises relative URLs and turns unsupported embeds into links.
func (s *Sanitizer) Prepare(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<div id=\"__root\">" + raw + "</div>"))
	if err != nil {
		return raw
	}
	doc.Find("iframe").Each(func(_ int, sel *goquery.Selection) {
		src, _ := sel.Attr("src")
		abs := resolveContentURL(src)
		if abs == "" {
			sel.Remove()
			return
		}
		u, err := url.Parse(abs)
		if err != nil || !allowedEmbedHosts[strings.ToLower(u.Host)] {
			link := "<a href=\"" + escapeAttr(abs) + "\">[嵌入内容] " + escapeAttr(abs) + "</a>"
			sel.ReplaceWithHtml(link)
			return
		}
		sel.SetAttr("src", abs)
	})
	doc.Find("img").Each(func(_ int, sel *goquery.Selection) {
		if src, ok := sel.Attr("src"); ok {
			sel.SetAttr("src", resolveContentURL(src))
		}
	})
	doc.Find("a").Each(func(_ int, sel *goquery.Selection) {
		if href, ok := sel.Attr("href"); ok {
			sel.SetAttr("href", resolveContentURL(href))
		}
	})
	inner, err := doc.Find("#__root").Html()
	if err != nil {
		return raw
	}
	return inner
}

// Sanitize prepares then applies the allow-list.
func (s *Sanitizer) Sanitize(raw string) string {
	return strings.TrimSpace(s.p.Sanitize(s.Prepare(raw)))
}

// Text returns plain text for search and AI prompts.
func (s *Sanitizer) Text(raw string) string {
	return HTMLText(s.Sanitize(raw))
}

func resolveContentURL(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.HasPrefix(ref, "#") {
		return ref
	}
	if strings.HasPrefix(ref, "//") {
		return "https:" + ref
	}
	u, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	if u.IsAbs() {
		if u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "mailto" {
			return ref
		}
		return ref
	}
	base, _ := url.Parse(baseURL)
	return base.ResolveReference(u).String()
}
