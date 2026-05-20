package rename

import (
	"path/filepath"
	"testing"
)

func TestStripEpisodeSuffix(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{"xxx 02", "xxx"},
		{"Show Name - 12", "Show Name"},
		{"Show [03]", "Show"},
		{"Anime S01E05", "Anime"},
		{"第04集 标题", "第04集 标题"},
		{"plain title", "plain title"},
	}
	for _, tc := range cases {
		got := StripEpisodeSuffix(tc.in)
		if got != tc.want {
			t.Fatalf("%q => %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestBuildScrapeFilename(t *testing.T) {
	t.Parallel()
	got := BuildScrapeFilename("/data/anime/xxx 02.mp4", 1, 4)
	want := filepath.Join("/data/anime", "xxx S01E04.mp4")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFinalEpisode(t *testing.T) {
	t.Parallel()
	got, err := FinalEpisode(2, 2)
	if err != nil || got != 4 {
		t.Fatalf("FinalEpisode(2,2) = %d, %v", got, err)
	}
	_, err = FinalEpisode(1, -2)
	if err == nil {
		t.Fatal("expected error for negative result")
	}
}
