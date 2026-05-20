package downloader

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func TestTellStatusReturnsStructuredRPCError(t *testing.T) {
	client := NewAria2Client("https://aria2.local/jsonrpc", "")
	client.httpClient = &http.Client{Transport: downloaderRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		body, err := json.Marshal(jsonRPCResponse{ID: "1", Error: &jsonRPCError{Code: 1, Message: "GID abcdef1234567890 is not found"}})
		if err != nil {
			t.Fatalf("marshal response: %v", err)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}

	_, err := client.TellStatus(context.Background(), "abcdef1234567890")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var rpcErr *Aria2RPCError
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected *Aria2RPCError, got %T (%v)", err, err)
	}
	if rpcErr.Code != 1 {
		t.Fatalf("rpcErr.Code = %d, want 1", rpcErr.Code)
	}
	if !IsGIDNotFound(err) {
		t.Fatalf("IsGIDNotFound should return true for %q", err)
	}
}

func TestIsGIDNotFound(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "plain error", err: errors.New("network unreachable"), want: false},
		{name: "rpc other error", err: &Aria2RPCError{Code: 2, Message: "Unauthorized"}, want: false},
		{name: "rpc GID not found", err: &Aria2RPCError{Code: 1, Message: "GID abc is not found"}, want: true},
		{name: "wrapped GID not found", err: fmt.Errorf("sync: %w", &Aria2RPCError{Code: 1, Message: "GID abc is not found"}), want: true},
		{name: "mixed case", err: &Aria2RPCError{Code: 1, Message: "No Such GID found"}, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsGIDNotFound(tc.err); got != tc.want {
				t.Fatalf("IsGIDNotFound(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
