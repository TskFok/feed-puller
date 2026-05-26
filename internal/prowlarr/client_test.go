package prowlarr

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_SearchMovies(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("X-Api-Key"); got != "secret" {
			t.Fatalf("unexpected api key %q", got)
		}
		q := r.URL.Query()
		if q.Get("query") != "inception" {
			t.Fatalf("unexpected query %q", q.Get("query"))
		}
		if q.Get("type") != "moviesearch" {
			t.Fatalf("unexpected type %q", q.Get("type"))
		}
		if q.Get("indexerIds") != "-2" {
			t.Fatalf("unexpected indexerIds %q", q.Get("indexerIds"))
		}
		if q.Get("categories") != "2000" {
			t.Fatalf("unexpected categories %q", q.Get("categories"))
		}
		if q.Get("limit") != "50" {
			t.Fatalf("unexpected limit %q", q.Get("limit"))
		}
		if q.Get("offset") != "10" {
			t.Fatalf("unexpected offset %q", q.Get("offset"))
		}
		_, _ = w.Write([]byte(`[{"guid":"g1","title":"Inception","protocol":"torrent","infoHash":"abc"}]`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret")
	releases, err := client.SearchMovies(context.Background(), SearchInput{
		Query:  "inception",
		Limit:  50,
		Offset: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 1 || releases[0].GUID != "g1" {
		t.Fatalf("unexpected releases: %+v", releases)
	}
}

func TestClient_SearchMovies_EmptyQuery(t *testing.T) {
	t.Parallel()
	client := NewClient("http://127.0.0.1:9696", "secret")
	_, err := client.SearchMovies(context.Background(), SearchInput{})
	if err == nil || !strings.Contains(err.Error(), "搜索关键词") {
		t.Fatalf("expected query error, got %v", err)
	}
}

func TestClient_TestConnection(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/system/status" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"version":"1.0.0"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret")
	if err := client.TestConnection(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestClient_TestConnection_Unauthorized(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`unauthorized`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad")
	err := client.TestConnection(context.Background())
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected 401 error, got %v", err)
	}
}
