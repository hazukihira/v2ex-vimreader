package v2ex

import (
	"fmt"
	"net/url"
	"strings"
)

// SiteURL joins a path with the v2ex.com origin.
func SiteURL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return baseURL + path
}

// HomeURL is the V2EX home page with the all-nodes tab.
func HomeURL() string { return baseURL + "/?tab=all" }

// NodeURL is one node list page.
func NodeURL(name string, page int) string {
	u := baseURL + "/go/" + url.PathEscape(name)
	if page > 1 {
		u += fmt.Sprintf("?p=%d", page)
	}
	return u
}

// RecentURL is one page of /recent.
func RecentURL(page int) string {
	if page <= 1 {
		return baseURL + "/recent"
	}
	return fmt.Sprintf("%s/recent?p=%d", baseURL, page)
}

// TopicURL is one page of a topic.
func TopicURL(id int64, page int) string {
	u := fmt.Sprintf("%s/t/%d", baseURL, id)
	if page > 1 {
		u += fmt.Sprintf("?p=%d", page)
	}
	return u
}

// TopicWebURL is the canonical browser URL of a topic.
func TopicWebURL(id int64) string { return fmt.Sprintf("%s/t/%d", baseURL, id) }

// NodeFeedURL is the JSON Feed of one node.
func NodeFeedURL(name string) string {
	return baseURL + "/feed/" + url.PathEscape(name) + ".json"
}

// IndexFeedURL is the site-wide Atom feed ordered by creation time.
func IndexFeedURL() string { return baseURL + "/index.xml" }

// ReadFeedURL is the second site-wide Atom feed.
func ReadFeedURL() string { return baseURL + "/read.xml" }

// PlanesURL is the node directory page.
func PlanesURL() string { return baseURL + "/planes" }

// MemberURL is a member profile URL.
func MemberURL(name string) string { return baseURL + "/member/" + url.PathEscape(name) }
