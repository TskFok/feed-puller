package rss

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
)

func TestFetcherUsesProxyOnlyWhenSubscriptionRequiresIt(t *testing.T) {
	directHits := 0
	proxyHits := 0
	fetcher := &Fetcher{
		proxyURL: "http://proxy.local:8080",
		directClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			directHits++
			return rssResponse(`<rss><channel><title>direct</title></channel></rss>`), nil
		})},
		proxyClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			proxyHits++
			return rssResponse(`<rss><channel><title>proxied</title></channel></rss>`), nil
		})},
	}

	direct, err := fetcher.Fetch(context.Background(), "https://example.com/feed.xml", false)
	if err != nil {
		t.Fatalf("direct Fetch returned error: %v", err)
	}
	if direct.Title != "direct" {
		t.Fatalf("direct title = %q", direct.Title)
	}

	proxied, err := fetcher.Fetch(context.Background(), "https://example.com/feed.xml", true)
	if err != nil {
		t.Fatalf("proxied Fetch returned error: %v", err)
	}
	if proxied.Title != "proxied" {
		t.Fatalf("proxied title = %q", proxied.Title)
	}

	if directHits != 1 {
		t.Fatalf("directHits = %d, want 1", directHits)
	}
	if proxyHits != 1 {
		t.Fatalf("proxyHits = %d, want 1", proxyHits)
	}
}

func TestFetcherReturnsConfigurationErrorWhenProxyRequiredButMissing(t *testing.T) {
	fetcher, err := NewFetcher("")
	if err != nil {
		t.Fatalf("NewFetcher returned error: %v", err)
	}

	_, err = fetcher.Fetch(context.Background(), "https://example.com/feed.xml", true)
	if err == nil {
		t.Fatal("expected error when proxy is required but not configured")
	}
}

func TestFetcherProbeContentLength(t *testing.T) {
	var sawHEAD bool
	fetcher := &Fetcher{
		directClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodHead {
				t.Fatalf("method = %q", r.Method)
			}
			sawHEAD = true
			return &http.Response{
				StatusCode:    http.StatusOK,
				ContentLength: 2048,
				Body:          io.NopCloser(bytes.NewBuffer(nil)),
				Header:        make(http.Header),
			}, nil
		})},
		proxyURL: "",
	}
	n, ok := fetcher.ProbeContentLength(context.Background(), "https://example.com/bin", false)
	if !ok || n != 2048 {
		t.Fatalf("ProbeContentLength = (%d, %v)", n, ok)
	}
	if !sawHEAD {
		t.Fatal("expected HEAD request")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func rssResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}
