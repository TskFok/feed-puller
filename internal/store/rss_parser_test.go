package store

import (
	"testing"

	"feed-puller/internal/rss"
)

func TestValidateSubscriptionRejectsUnknownParser(t *testing.T) {
	err := validateSubscription(Subscription{
		Name:                "x",
		FeedURL:             "https://example.com/feed",
		DownloadDir:         "/data",
		PollIntervalMinutes: 30,
		RSSParser:           "unknown",
	})
	if err == nil {
		t.Fatal("expected error for unknown parser")
	}
}

func TestValidateSubscriptionAcceptsMikanParser(t *testing.T) {
	err := validateSubscription(Subscription{
		Name:                "x",
		FeedURL:             "https://mikanani.me/RSS/Bangumi?bangumiId=1",
		DownloadDir:         "/data",
		PollIntervalMinutes: 30,
		RSSParser:           rss.ParserMikan,
	})
	if err != nil {
		t.Fatal(err)
	}
}
