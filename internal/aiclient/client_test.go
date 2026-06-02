package aiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatCompletionsURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{"https://api.openai.com/v1", "https://api.openai.com/v1/chat/completions"},
		{"https://api.openai.com/v1/", "https://api.openai.com/v1/chat/completions"},
		{"https://api.openai.com/v1/chat/completions", "https://api.openai.com/v1/chat/completions"},
		{"https://proxy.example.com", "https://proxy.example.com/v1/chat/completions"},
	}
	for _, tc := range cases {
		got, err := chatCompletionsURL(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("%q => %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestTestConnection_DefaultPayloadOmitsOptionalParameters(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-ok" {
			t.Fatalf("missing auth header")
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if _, ok := payload["temperature"]; ok {
			t.Fatalf("temperature should be omitted by default: %#v", payload)
		}
		if payload["max_tokens"] != float64(1) {
			t.Fatalf("max_tokens = %v, want 1", payload["max_tokens"])
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"chatcmpl-test"}`))
	}))
	t.Cleanup(srv.Close)

	if err := TestConnection(context.Background(), srv.URL+"/v1", "sk-ok", "gpt-test", ""); err != nil {
		t.Fatal(err)
	}
}

func TestTestConnection_MergesRequestOptions(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		thinking, ok := payload["thinking"].(map[string]any)
		if !ok || thinking["type"] != "disabled" {
			t.Fatalf("thinking option not merged: %#v", payload)
		}
		if payload["temperature"] != 0.6 {
			t.Fatalf("temperature = %v, want 0.6", payload["temperature"])
		}
		if payload["model"] != "gpt-test" {
			t.Fatalf("model = %v, want gpt-test", payload["model"])
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"chatcmpl-test"}`))
	}))
	t.Cleanup(srv.Close)

	options := `{"thinking":{"type":"disabled"},"temperature":0.6}`
	if err := TestConnection(context.Background(), srv.URL+"/v1", "sk-ok", "gpt-test", options); err != nil {
		t.Fatal(err)
	}
}

func TestTestConnection_HTTPError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "invalid key", http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	err := TestConnection(context.Background(), srv.URL+"/v1", "sk-bad", "gpt-test", "")
	if err == nil {
		t.Fatal("expected error")
	}
}
