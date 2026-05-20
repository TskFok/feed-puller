package downloader

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestFirstFollowedByGID(t *testing.T) {
	t.Parallel()
	status := map[string]any{
		"followedBy": []any{"abcdef1234567890"},
	}
	if got := FirstFollowedByGID(status); got != "abcdef1234567890" {
		t.Fatalf("got %q", got)
	}
}

func TestFollowingGID(t *testing.T) {
	t.Parallel()
	status := map[string]any{"following": "1111222233334444"}
	if got := FollowingGID(status); got != "1111222233334444" {
		t.Fatalf("got %q", got)
	}
}

func TestTellStatusEffective_FollowsChain(t *testing.T) {
	calls := 0
	client := NewAria2Client("http://aria2.test/jsonrpc", "")
	client.httpClient = &http.Client{Transport: downloaderRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		var req jsonRPCRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		gid, _ := req.Params[len(req.Params)-1].(string)
		result := map[string]any{"status": "active", "completedLength": "10", "totalLength": "100"}
		if gid == "meta-gid" {
			result["followedBy"] = []any{"real-gid"}
			result["status"] = "complete"
		}
		if gid == "real-gid" {
			result["completedLength"] = "50"
		}
		body, _ := json.Marshal(jsonRPCResponse{ID: "1", Result: result})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body)), Header: http.Header{}}, nil
	})}

	effective, status, err := client.TellStatusEffective(context.Background(), "meta-gid")
	if err != nil {
		t.Fatal(err)
	}
	if effective != "real-gid" {
		t.Fatalf("effective = %q, want real-gid", effective)
	}
	if calls != 2 {
		t.Fatalf("tellStatus calls = %d, want 2", calls)
	}
	if status["status"] != "active" {
		t.Fatalf("status = %v", status["status"])
	}
}
