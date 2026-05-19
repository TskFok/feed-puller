package rss

import (
	"testing"
)

func TestValidateKeywordPatterns_InvalidInclude(t *testing.T) {
	t.Parallel()
	err := ValidateKeywordPatterns(`(unclosed`, ``)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFilterFeedItems_ExcludeWins(t *testing.T) {
	t.Parallel()
	items := []FeedItem{
		{Title: "Alpha Episode", Link: "http://a"},
		{Title: "Beta Special", Link: "http://b"},
	}
	out, err := FilterFeedItems(items, ``, `Special`)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Title != "Alpha Episode" {
		t.Fatalf("got %#v", out)
	}
}

func TestFilterFeedItems_IncludeRequired(t *testing.T) {
	t.Parallel()
	items := []FeedItem{
		{Title: "Foo Bar", Link: "http://x"},
		{Title: "Other", Link: "http://y"},
	}
	out, err := FilterFeedItems(items, `Foo`, ``)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Title != "Foo Bar" {
		t.Fatalf("got %#v", out)
	}
}

func TestFilterFeedItems_IncludeAndExclude(t *testing.T) {
	t.Parallel()
	items := []FeedItem{
		{Title: "Show 1080p", Link: "http://1"},
		{Title: "Show 720p", Link: "http://2"},
		{Title: "Other", Link: "http://3"},
	}
	out, err := FilterFeedItems(items, `Show`, `720`)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Title != "Show 1080p" {
		t.Fatalf("got %#v", out)
	}
}

func TestFilterFeedItems_MultilinePatterns(t *testing.T) {
	t.Parallel()
	items := []FeedItem{
		{Title: "AAA", Link: ""},
		{Title: "BBB", Link: ""},
		{Title: "CCC", Link: ""},
	}
	raw := "AAA\nBBB"
	out, err := FilterFeedItems(items, raw, ``)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("got %#v", out)
	}
}
