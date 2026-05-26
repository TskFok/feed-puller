package prowlarr

import (
	"testing"
	"time"
)

func TestNormalizeSearchQuery_IMDb(t *testing.T) {
	t.Parallel()
	got := NormalizeSearchQuery("tt1375666", SearchTypeMovie)
	if got != "{ImdbId:tt1375666}" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeSearchQuery_TVDB(t *testing.T) {
	t.Parallel()
	got := NormalizeSearchQuery("80348", SearchTypeTV)
	if got != "{TvdbId:80348}" {
		t.Fatalf("got %q", got)
	}
}

func TestSortReleases_BySize(t *testing.T) {
	t.Parallel()
	releases := []Release{
		{GUID: "a", Size: 100},
		{GUID: "b", Size: 300},
		{GUID: "c", Size: 200},
	}
	SortReleases(releases, SortBySize)
	if releases[0].GUID != "b" || releases[2].GUID != "a" {
		t.Fatalf("unexpected order: %+v", releases)
	}
}

func TestSortReleases_ByDate(t *testing.T) {
	t.Parallel()
	now := time.Now()
	releases := []Release{
		{GUID: "old", PublishDate: now.Add(-2 * time.Hour)},
		{GUID: "new", PublishDate: now},
	}
	SortReleases(releases, SortByDate)
	if releases[0].GUID != "new" {
		t.Fatalf("expected newest first")
	}
}
