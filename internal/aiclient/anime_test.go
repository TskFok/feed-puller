package aiclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractAnimeInfo_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"anime_name\":\"鬼灭之刃\",\"episode\":3}"}}]}`))
	}))
	t.Cleanup(srv.Close)

	got, err := ExtractAnimeInfo(context.Background(), srv.URL+"/v1", "sk-test", "gpt-test", "xxx 03.mp4", "第3话")
	if err != nil {
		t.Fatal(err)
	}
	if got.AnimeName != "鬼灭之刃" || got.Episode != 3 {
		t.Fatalf("got %+v", got)
	}
}

func TestExtractAnimeInfo_PartialEpisodeOnly(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"episode\": 2}"}}]}`))
	}))
	t.Cleanup(srv.Close)

	got, err := ExtractAnimeInfo(context.Background(), srv.URL+"/v1", "sk-test", "gpt-test", "番剧 第02话.mp4", "第2话")
	if err != nil {
		t.Fatal(err)
	}
	if got.Episode != 2 {
		t.Fatalf("got %+v", got)
	}
}

func TestParseAnimeInfo_InvalidEpisode(t *testing.T) {
	t.Parallel()
	_, err := parseAnimeInfo(`{"episode": 0}`)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildAnimeExtractPrompt(t *testing.T) {
	t.Parallel()
	prompt := BuildAnimeExtractPrompt("xxx 03.mp4", "第3话")
	if !strings.Contains(prompt, "xxx 03.mp4") || !strings.Contains(prompt, "第3话") {
		t.Fatalf("prompt = %q", prompt)
	}
}

func TestExtractAnimeInfoDetailed_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"anime_name\":\"鬼灭之刃\",\"episode\":3}"}}]}`))
	}))
	t.Cleanup(srv.Close)

	got, err := ExtractAnimeInfoDetailed(context.Background(), srv.URL+"/v1", "sk-test", "gpt-test", "xxx 03.mp4", "第3话")
	if err != nil {
		t.Fatal(err)
	}
	if got.Prompt == "" || got.RawResponse == "" {
		t.Fatalf("missing metadata: %+v", got)
	}
	if got.Info.AnimeName != "鬼灭之刃" || got.Info.Episode != 3 {
		t.Fatalf("got %+v", got.Info)
	}
}
