package rss

import "testing"

func TestParseFeedExtractsStandardDownloadTargets(t *testing.T) {
	input := []byte(`<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Example</title>
    <item>
      <guid>item-1</guid>
      <title>Enclosure item</title>
      <link>https://example.com/post/1</link>
      <enclosure url="https://cdn.example.com/file.torrent" type="application/x-bittorrent"/>
    </item>
    <item>
      <guid>item-2</guid>
      <title>Magnet item</title>
      <link>magnet:?xt=urn:btih:abcdef</link>
    </item>
  </channel>
</rss>`)

	feed, err := ParseFeed(input)
	if err != nil {
		t.Fatalf("ParseFeed returned error: %v", err)
	}

	if feed.Title != "Example" {
		t.Fatalf("feed title = %q, want Example", feed.Title)
	}
	if len(feed.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(feed.Items))
	}
	if got := feed.Items[0].DownloadURL; got != "https://cdn.example.com/file.torrent" {
		t.Fatalf("first download URL = %q", got)
	}
	if got := feed.Items[1].DownloadURL; got != "magnet:?xt=urn:btih:abcdef" {
		t.Fatalf("second download URL = %q", got)
	}
}

func TestItemDedupeKeyPrefersGUIDThenLinkThenDownloadURL(t *testing.T) {
	cases := []struct {
		name string
		item FeedItem
		want string
	}{
		{
			name: "guid",
			item: FeedItem{GUID: " item-1 ", Link: "https://example.com/post", DownloadURL: "https://example.com/file.torrent"},
			want: "guid:item-1",
		},
		{
			name: "link",
			item: FeedItem{Link: "HTTPS://Example.com/Post?b=2&a=1"},
			want: "link:https://example.com/Post?a=1&b=2",
		},
		{
			name: "download",
			item: FeedItem{DownloadURL: "magnet:?xt=urn:btih:abcdef"},
			want: "download:magnet:?xt=urn:btih:abcdef",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DedupeKey(tc.item); got != tc.want {
				t.Fatalf("DedupeKey() = %q, want %q", got, tc.want)
			}
		})
	}
}
