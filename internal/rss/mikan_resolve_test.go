package rss

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"testing"
)

func TestResolveMikanDownloadURLReturnsMagnet(t *testing.T) {
	torrent, err := os.ReadFile("testdata/mikan_sample.torrent")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	fetcher := &Fetcher{
		directClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "https://mikanani.me/Download/20260518/abc.torrent" {
				t.Fatalf("unexpected url %s", r.URL)
			}
			if r.Header.Get("Referer") != "https://mikanani.me/" {
				t.Fatalf("referer = %q", r.Header.Get("Referer"))
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(torrent)),
				Header:     make(http.Header),
			}, nil
		})},
	}

	magnet, err := ResolveMikanDownloadURL(context.Background(), fetcher, "https://mikanani.me/Download/20260518/abc.torrent", false)
	if err != nil {
		t.Fatalf("ResolveMikanDownloadURL: %v", err)
	}
	if magnet == "" || magnet[:8] != "magnet:?" {
		t.Fatalf("magnet = %q", magnet)
	}
}

func TestResolveMikanDownloadURLPassesThroughMagnet(t *testing.T) {
	raw := "magnet:?xt=urn:btih:ABCDEF"
	got, err := ResolveMikanDownloadURL(context.Background(), &Fetcher{}, raw, false)
	if err != nil || got != raw {
		t.Fatalf("got %q err %v", got, err)
	}
}
