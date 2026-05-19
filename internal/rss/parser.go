package rss

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

type Feed struct {
	Title string
	Items []FeedItem
}

type FeedItem struct {
	GUID        string
	Title       string
	Link        string
	DownloadURL string
	PublishedAt *time.Time
}

type rssDocument struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title string    `xml:"title"`
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	GUID        string         `xml:"guid"`
	Title       string         `xml:"title"`
	Link        string         `xml:"link"`
	PubDate     string         `xml:"pubDate"`
	Enclosures  []rssEnclosure `xml:"enclosure"`
	Description string         `xml:"description"`
}

type rssEnclosure struct {
	URL  string `xml:"url,attr"`
	Type string `xml:"type,attr"`
}

type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Title   string      `xml:"title"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	ID        string     `xml:"id"`
	Title     string     `xml:"title"`
	Updated   string     `xml:"updated"`
	Published string     `xml:"published"`
	Links     []atomLink `xml:"link"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

func ParseFeed(input []byte) (Feed, error) {
	return ParseFeedWithParser(input, ParserGeneric)
}

// ParseFeedWithParser 按解析器类型解析 RSS/Atom 内容。
func ParseFeedWithParser(input []byte, parser string) (Feed, error) {
	input = bytes.TrimSpace(input)
	if len(input) == 0 {
		return Feed{}, fmt.Errorf("订阅内容为空")
	}

	var probe struct {
		XMLName xml.Name
	}
	if err := xml.Unmarshal(input, &probe); err != nil {
		return Feed{}, fmt.Errorf("解析订阅 XML 失败: %w", err)
	}

	switch strings.ToLower(probe.XMLName.Local) {
	case "rss":
		if NormalizeParser(parser) == ParserMikan {
			return parseMikanRSS(input)
		}
		return parseRSS(input)
	case "feed":
		return parseAtom(input)
	default:
		return Feed{}, fmt.Errorf("不支持的订阅格式: %s", probe.XMLName.Local)
	}
}

func parseRSS(input []byte) (Feed, error) {
	var doc rssDocument
	if err := xml.Unmarshal(input, &doc); err != nil {
		return Feed{}, fmt.Errorf("解析 RSS 失败: %w", err)
	}

	feed := Feed{Title: strings.TrimSpace(doc.Channel.Title)}
	for _, item := range doc.Channel.Items {
		link := strings.TrimSpace(item.Link)
		candidates := append(enclosureURLs(item.Enclosures), link)
		downloadURL := firstDownloadURL(candidates...)
		feed.Items = append(feed.Items, FeedItem{
			GUID:        strings.TrimSpace(item.GUID),
			Title:       strings.TrimSpace(item.Title),
			Link:        link,
			DownloadURL: downloadURL,
			PublishedAt: parseFeedTime(item.PubDate),
		})
	}
	return feed, nil
}

func parseAtom(input []byte) (Feed, error) {
	var doc atomFeed
	if err := xml.Unmarshal(input, &doc); err != nil {
		return Feed{}, fmt.Errorf("解析 Atom 失败: %w", err)
	}

	feed := Feed{Title: strings.TrimSpace(doc.Title)}
	for _, entry := range doc.Entries {
		link := atomEntryLink(entry.Links, "alternate")
		downloadURL := atomEntryLink(entry.Links, "enclosure")
		if downloadURL == "" {
			downloadURL = firstDownloadURL(link)
		}
		if link == "" {
			link = downloadURL
		}
		publishedAt := parseFeedTime(entry.Published)
		if publishedAt == nil {
			publishedAt = parseFeedTime(entry.Updated)
		}
		feed.Items = append(feed.Items, FeedItem{
			GUID:        strings.TrimSpace(entry.ID),
			Title:       strings.TrimSpace(entry.Title),
			Link:        link,
			DownloadURL: downloadURL,
			PublishedAt: publishedAt,
		})
	}
	return feed, nil
}

func DedupeKey(item FeedItem) string {
	if guid := strings.TrimSpace(item.GUID); guid != "" {
		return "guid:" + guid
	}
	if link := normalizeHTTPURL(item.Link); link != "" {
		return "link:" + link
	}
	if downloadURL := strings.TrimSpace(item.DownloadURL); downloadURL != "" {
		return "download:" + downloadURL
	}
	return ""
}

func enclosureURLs(enclosures []rssEnclosure) []string {
	urls := make([]string, 0, len(enclosures))
	for _, enclosure := range enclosures {
		if enclosure.URL != "" {
			urls = append(urls, enclosure.URL)
		}
	}
	return urls
}

func firstDownloadURL(candidates ...string) string {
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if isSupportedDownloadURL(candidate) {
			return candidate
		}
	}
	return ""
}

func isSupportedDownloadURL(raw string) bool {
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "magnet:") {
		return true
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" {
		return false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "ftp", "sftp":
		return true
	default:
		return false
	}
}

func atomEntryLink(links []atomLink, rel string) string {
	for _, link := range links {
		linkRel := link.Rel
		if linkRel == "" {
			linkRel = "alternate"
		}
		if strings.EqualFold(linkRel, rel) && strings.TrimSpace(link.Href) != "" {
			return strings.TrimSpace(link.Href)
		}
	}
	return ""
}

func normalizeHTTPURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" {
		return raw
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	query := parsed.Query()
	if len(query) > 0 {
		keys := make([]string, 0, len(query))
		for key := range query {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		ordered := url.Values{}
		for _, key := range keys {
			values := append([]string(nil), query[key]...)
			sort.Strings(values)
			for _, value := range values {
				ordered.Add(key, value)
			}
		}
		parsed.RawQuery = ordered.Encode()
	}
	return parsed.String()
}

func parseFeedTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
		time.RFC3339Nano,
		"Mon, 02 Jan 2006 15:04:05 -0700",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return &parsed
		}
	}
	return nil
}
