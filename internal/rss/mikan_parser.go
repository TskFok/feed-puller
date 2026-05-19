package rss

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

type mikanRSSDocument struct {
	XMLName xml.Name       `xml:"rss"`
	Channel mikanRSSChannel `xml:"channel"`
}

type mikanRSSChannel struct {
	Title string         `xml:"title"`
	Items []mikanRSSItem `xml:"item"`
}

type mikanRSSItem struct {
	GUID        string         `xml:"guid"`
	Title       string         `xml:"title"`
	Link        string         `xml:"link"`
	PubDate     string         `xml:"pubDate"`
	Enclosures  []rssEnclosure `xml:"enclosure"`
	Torrent     mikanTorrent   `xml:"torrent"`
}

type mikanTorrent struct {
	PubDate string `xml:"pubDate"`
}

func parseMikanRSS(input []byte) (Feed, error) {
	var doc mikanRSSDocument
	if err := xml.Unmarshal(input, &doc); err != nil {
		return Feed{}, fmt.Errorf("解析 Mikan RSS 失败: %w", err)
	}

	feed := Feed{Title: strings.TrimSpace(doc.Channel.Title)}
	for _, item := range doc.Channel.Items {
		link := strings.TrimSpace(item.Link)
		candidates := append(enclosureURLs(item.Enclosures), link)
		downloadURL := firstDownloadURL(candidates...)
		publishedAt := parseFeedTime(item.PubDate)
		if publishedAt == nil {
			publishedAt = parseMikanTorrentTime(item.Torrent.PubDate)
		}
		feed.Items = append(feed.Items, FeedItem{
			GUID:        strings.TrimSpace(item.GUID),
			Title:       strings.TrimSpace(item.Title),
			Link:        link,
			DownloadURL: downloadURL,
			PublishedAt: publishedAt,
		})
	}
	return feed, nil
}

func parseMikanTorrentTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	layouts := []string{
		"2006-01-02T15:04:05.999999",
		"2006-01-02T15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return &parsed
		}
	}
	return nil
}
