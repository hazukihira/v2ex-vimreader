package v2ex

import (
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	reTopicPath     = regexp.MustCompile(`/t/(\d+)`)
	reReplyAnchor   = regexp.MustCompile(`#reply(\d+)`)
	reTopicIDScript = regexp.MustCompile(`topicId\s*=\s*(\d+)`)
	reNodeTotal     = regexp.MustCompile(`Topics\s+\d+\s+to\s+\d+\s+of\s+(\d+)`)
	reRecentPages   = regexp.MustCompile(`Recent Topics\s+\d+\s*/\s*(\d+)`)
	reViews         = regexp.MustCompile(`([0-9,]+)\s*(?:views|次点击)`)
	reNodePrefix    = regexp.MustCompile(`^\[([^\]]{1,60})\]\s*(.*)$`)
)

func parseUnixTime(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	layouts := []string{
		"2006-01-02 15:04:05 -07:00",
		time.RFC3339,
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Unix()
		}
	}
	return 0
}

func atoiSafe(s string) int {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func topicIDFromHref(href string) int64 {
	m := reTopicPath.FindStringSubmatch(href)
	if m == nil {
		return 0
	}
	id, _ := strconv.ParseInt(m[1], 10, 64)
	return id
}

func replyCountFromHref(href string) (int, bool) {
	m := reReplyAnchor.FindStringSubmatch(href)
	if m == nil {
		return 0, false
	}
	return atoiSafe(m[1]), true
}

// HTMLText converts a small HTML fragment to plain text.
func HTMLText(in string) string {
	if in == "" {
		return ""
	}
	var b strings.Builder
	tokenizer := newHTMLTokenizer(in)
	for {
		tt := tokenizer.Next()
		switch tt {
		case htmlTokenizerError:
			return normalizeText(b.String())
		case htmlTokenizerText:
			b.WriteString(tokenizer.Text())
		case htmlTokenizerStart, htmlTokenizerEnd:
			name := strings.ToLower(tokenizer.TagName())
			switch name {
			case "br", "p", "div", "li", "tr", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote", "pre", "table":
				b.WriteString("\n")
			}
		}
	}
}

func normalizeText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	var out []string
	blank := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if !blank && len(out) > 0 {
				out = append(out, "")
			}
			blank = true
			continue
		}
		blank = false
		out = append(out, line)
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

func escapeAttr(s string) string { return html.EscapeString(s) }
