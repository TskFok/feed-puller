package aiclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestModelsURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{"https://api.openai.com/v1", "https://api.openai.com/v1/models"},
		{"https://api.openai.com/v1/", "https://api.openai.com/v1/models"},
		{"https://api.openai.com/v1/models", "https://api.openai.com/v1/models"},
		{"https://proxy.example.com", "https://proxy.example.com/v1/models"},
		{"https://api.openai.com/v1/chat/completions", "https://api.openai.com/v1/models"},
	}
	for _, tc := range cases {
		got, err := modelsURL(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("%q => %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestListModels_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-ok" {
			t.Fatalf("missing auth header")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4o-mini"},{"id":"gpt-4o"},{"id":"gpt-4o-mini"}]}`))
	}))
	t.Cleanup(srv.Close)

	models, err := ListModels(context.Background(), srv.URL+"/v1", "sk-ok")
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0] != "gpt-4o" || models[1] != "gpt-4o-mini" {
		t.Fatalf("unexpected models: %#v", models)
	}
}

func TestListModels_AllowsEmptyAPIKey(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Fatalf("unexpected auth header")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"id":"llama3.2"}]}`))
	}))
	t.Cleanup(srv.Close)

	models, err := ListModels(context.Background(), srv.URL+"/v1", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0] != "llama3.2" {
		t.Fatalf("unexpected models: %#v", models)
	}
}

func TestListModels_HTTPError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "invalid key", http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	_, err := ListModels(context.Background(), srv.URL+"/v1", "sk-bad")
	if err == nil {
		t.Fatal("expected error")
	}
}
