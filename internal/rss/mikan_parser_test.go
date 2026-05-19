package rss

import "testing"

func TestParseMikanFeedUsesEnclosureAndTorrentPubDate(t *testing.T) {
	input := []byte(`<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0">
  <channel>
    <title>Mikan Demo</title>
    <item>
      <guid isPermaLink="false">episode-06</guid>
      <link>https://mikanani.me/Home/Episode/abc</link>
      <title>Demo - 06</title>
      <pubDate>Mon, 18 May 2026 01:00:00 +0000</pubDate>
      <enclosure type="application/x-bittorrent" length="100" url="https://mikanani.me/Download/20260518/abc.torrent"/>
      <torrent xmlns="https://mikanani.me/0.1/">
        <pubDate>2026-05-18T01:00:51.216428</pubDate>
      </torrent>
    </item>
  </channel>
</rss>`)

	feed, err := ParseFeedWithParser(input, ParserMikan)
	if err != nil {
		t.Fatalf("ParseFeedWithParser: %v", err)
	}
	if len(feed.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(feed.Items))
	}
	item := feed.Items[0]
	if item.DownloadURL != "https://mikanani.me/Download/20260518/abc.torrent" {
		t.Fatalf("download_url = %q", item.DownloadURL)
	}
	if item.PublishedAt == nil {
		t.Fatal("expected published_at")
	}
}

func TestIsMikanTorrentURL(t *testing.T) {
	if !IsMikanTorrentURL("https://mikanani.me/Download/20260518/abc.torrent") {
		t.Fatal("expected mikan torrent url")
	}
	if IsMikanTorrentURL("https://example.com/file.torrent") {
		t.Fatal("unexpected match for non-mikan host")
	}
}
