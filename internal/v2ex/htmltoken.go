package v2ex

import (
	"strings"

	"golang.org/x/net/html"
)

type htmlTokenizer struct {
	z *html.Tokenizer
}

const (
	htmlTokenizerError = iota
	htmlTokenizerText
	htmlTokenizerStart
	htmlTokenizerEnd
)

func newHTMLTokenizer(in string) *htmlTokenizer {
	return &htmlTokenizer{z: html.NewTokenizer(strings.NewReader(in))}
}

func (t *htmlTokenizer) Next() int {
	tt := t.z.Next()
	switch tt {
	case html.ErrorToken:
		return htmlTokenizerError
	case html.TextToken:
		return htmlTokenizerText
	case html.StartTagToken, html.SelfClosingTagToken:
		return htmlTokenizerStart
	case html.EndTagToken:
		return htmlTokenizerEnd
	default:
		return htmlTokenizerText
	}
}

func (t *htmlTokenizer) Text() string { return t.z.Token().Data }

func (t *htmlTokenizer) TagName() string {
	name, _ := t.z.TagName()
	return string(name)
}
