package prowlarr

import "testing"

func TestResolveTorrentURL_PrefersInfoHash(t *testing.T) {
	t.Parallel()
	url, err := ResolveTorrentURL(Release{
		InfoHash:    "abc123",
		DownloadURL: "http://example.com/torrent",
	})
	if err != nil {
		t.Fatal(err)
	}
	if url != "magnet:?xt=urn:btih:abc123" {
		t.Fatalf("got %q", url)
	}
}

func TestResolveTorrentURL_FallsBackToDownloadURL(t *testing.T) {
	t.Parallel()
	url, err := ResolveTorrentURL(Release{DownloadURL: "http://example.com/torrent"})
	if err != nil {
		t.Fatal(err)
	}
	if url != "http://example.com/torrent" {
		t.Fatalf("got %q", url)
	}
}

func TestResolveTorrentURL_Empty(t *testing.T) {
	t.Parallel()
	_, err := ResolveTorrentURL(Release{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFilterTorrentReleases(t *testing.T) {
	t.Parallel()
	filtered := FilterTorrentReleases([]Release{
		{Protocol: "torrent", Title: "a"},
		{Protocol: "usenet", Title: "b"},
		{Protocol: "Torrent", Title: "c"},
	})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(filtered))
	}
}
