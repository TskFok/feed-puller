package tmdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetMovieDetails_ByID(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/3/movie/27205") {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"title":"Inception","release_date":"2010-07-16"}`))
	}))
	defer server.Close()

	client := NewClient("secret")
	client.baseURL = server.URL + "/3"
	details, err := client.GetMovieDetails(context.Background(), 27205, 0)
	if err != nil {
		t.Fatal(err)
	}
	if details.Title != "Inception" || details.Year != 2010 {
		t.Fatalf("unexpected details: %+v", details)
	}
}

func TestFormatIMDbID(t *testing.T) {
	t.Parallel()
	if got := FormatIMDbID(1375666); got != "tt1375666" {
		t.Fatalf("got %q", got)
	}
}

func TestParseYear(t *testing.T) {
	t.Parallel()
	if parseYear("2010-07-16") != 2010 {
		t.Fatal("expected 2010")
	}
}

func TestGetMovieDetails_Disabled(t *testing.T) {
	t.Parallel()
	client := NewClient("")
	_, err := client.GetMovieDetails(context.Background(), 1, 0)
	if err == nil {
		t.Fatal("expected error")
	}
}
