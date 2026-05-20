package rename

import (
	"path/filepath"
	"testing"
)

func TestResolveScrapeTarget_WithAI(t *testing.T) {
	t.Parallel()
	from := "/data/anime/xxx 02.mp4"
	got, err := ResolveScrapeTarget(ScrapeInput{
		FilePath:           from,
		Filename:           "xxx 02.mp4",
		SubscriptionSeason: 1,
		EpisodeOffset:      2,
		AI:                 &AnimeExtract{AnimeName: "鬼灭之刃", Episode: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/data/anime", "鬼灭之刃 S01E04.mp4")
	if got.Path != want {
		t.Fatalf("path = %q, want %q", got.Path, want)
	}
	if got.Season != 1 || got.Episode != 4 {
		t.Fatalf("season/episode = %d/%d", got.Season, got.Episode)
	}
}

func TestResolveScrapeTarget_LocalFallback(t *testing.T) {
	t.Parallel()
	from := "/data/番剧 第02话.mp4"
	got, err := ResolveScrapeTarget(ScrapeInput{
		FilePath:           from,
		Filename:           "番剧 第02话.mp4",
		Title:              "第2话",
		SubscriptionSeason: 1,
		EpisodeOffset:      0,
		LocalEpisode:       2,
		LocalEpisodeOK:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/data", "番剧 第02话 S01E02.mp4")
	if got.Path != want {
		t.Fatalf("path = %q, want %q", got.Path, want)
	}
}

func TestResolveScrapeTarget_AINameFallbackToFilename(t *testing.T) {
	t.Parallel()
	from := "/data/番剧 第02话.mp4"
	got, err := ResolveScrapeTarget(ScrapeInput{
		FilePath:           from,
		Filename:           "番剧 第02话.mp4",
		SubscriptionSeason: 1,
		EpisodeOffset:      0,
		AI:                 &AnimeExtract{Episode: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/data", "番剧 第02话 S01E02.mp4")
	if got.Path != want {
		t.Fatalf("path = %q, want %q", got.Path, want)
	}
}
