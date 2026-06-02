package aiclient

import (
	"context"
	"encoding/json"
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

	got, err := ExtractAnimeInfo(context.Background(), srv.URL+"/v1", "sk-test", "gpt-test", "", "xxx 03.mp4", "第3话")
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

	got, err := ExtractAnimeInfo(context.Background(), srv.URL+"/v1", "sk-test", "gpt-test", "", "番剧 第02话.mp4", "第2话")
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

	got, err := ExtractAnimeInfoDetailed(context.Background(), srv.URL+"/v1", "sk-test", "gpt-test", "", "xxx 03.mp4", "第3话")
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

func TestExtractAnimeInfoDetailed_DefaultPayloadOmitsOptionalParameters(t *testing.T) {
	t.Parallel()
	var requestBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"anime_name\":\"鬼灭之刃\",\"episode\":3}"}}]}`))
	}))
	t.Cleanup(srv.Close)

	_, err := ExtractAnimeInfoDetailed(context.Background(), srv.URL+"/v1", "sk-test", "gpt-test", "", "xxx 03.mp4", "第3话")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := requestBody["temperature"]; ok {
		t.Fatalf("temperature should be omitted by default: %#v", requestBody)
	}
	if requestBody["max_tokens"] != float64(128) {
		t.Fatalf("max_tokens = %v, want 128", requestBody["max_tokens"])
	}
}

func TestExtractAnimeInfoDetailed_MergesRequestOptions(t *testing.T) {
	t.Parallel()
	var requestBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"anime_name\":\"鬼灭之刃\",\"episode\":3}"}}]}`))
	}))
	t.Cleanup(srv.Close)

	options := `{"thinking":{"type":"disabled"},"top_p":0.95}`
	_, err := ExtractAnimeInfoDetailed(context.Background(), srv.URL+"/v1", "sk-test", "gpt-test", options, "xxx 03.mp4", "第3话")
	if err != nil {
		t.Fatal(err)
	}
	thinking, ok := requestBody["thinking"].(map[string]any)
	if !ok || thinking["type"] != "disabled" {
		t.Fatalf("thinking option not merged: %#v", requestBody)
	}
	if requestBody["top_p"] != 0.95 {
		t.Fatalf("top_p = %v, want 0.95", requestBody["top_p"])
	}
}
