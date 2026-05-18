package downloader

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestAria2AddURIIncludesTokenURLAndDirectory(t *testing.T) {
	var payload jsonRPCRequest
	client := NewAria2Client("https://aria2.local/jsonrpc", "secret")
	client.httpClient = &http.Client{Transport: downloaderRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		body, err := json.Marshal(jsonRPCResponse{ID: payload.ID, Result: "gid-123"})
		if err != nil {
			t.Fatalf("marshal response: %v", err)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}

	gid, err := client.AddURI(context.Background(), "magnet:?xt=urn:btih:abcdef", "/downloads/show")
	if err != nil {
		t.Fatalf("AddURI returned error: %v", err)
	}
	if gid != "gid-123" {
		t.Fatalf("gid = %q, want gid-123", gid)
	}
	if payload.Method != "aria2.addUri" {
		t.Fatalf("method = %q", payload.Method)
	}
	if got := payload.Params[0]; got != "token:secret" {
		t.Fatalf("first param = %#v, want token", got)
	}
	uris, ok := payload.Params[1].([]any)
	if !ok || len(uris) != 1 || uris[0] != "magnet:?xt=urn:btih:abcdef" {
		t.Fatalf("URI param = %#v", payload.Params[1])
	}
	options, ok := payload.Params[2].(map[string]any)
	if !ok || options["dir"] != "/downloads/show" {
		t.Fatalf("options = %#v", payload.Params[2])
	}
}

type downloaderRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn downloaderRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
