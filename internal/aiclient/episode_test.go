package aiclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractEpisode_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"episode\": 2}"}}]}`))
	}))
	t.Cleanup(srv.Close)

	got, err := ExtractEpisode(context.Background(), srv.URL+"/v1", "sk-test", "gpt-test", "xxx 02.mp4", "第2话")
	if err != nil {
		t.Fatal(err)
	}
	if got != 2 {
		t.Fatalf("episode = %d, want 2", got)
	}
}

func TestExtractEpisode_InvalidResponse(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"episode\": 0}"}}]}`))
	}))
	t.Cleanup(srv.Close)

	_, err := ExtractEpisode(context.Background(), srv.URL+"/v1", "sk-test", "gpt-test", "unknown.mp4", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseEpisodeNumber(t *testing.T) {
	t.Parallel()
	got, err := parseEpisodeNumber(`{"episode": 12}`)
	if err != nil || got != 12 {
		t.Fatalf("parseEpisodeNumber = %d, %v", got, err)
	}
}
