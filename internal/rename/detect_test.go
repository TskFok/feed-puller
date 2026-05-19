package rename

import "testing"

func TestDetectEpisodeLocally(t *testing.T) {
	t.Parallel()
	cases := []struct {
		file  string
		title string
		want  int
	}{
		{"xxx 02.mp4", "", 2},
		{"show.mp4", "第12集", 12},
		{"Anime S01E05.mkv", "", 5},
	}
	for _, tc := range cases {
		got, ok := DetectEpisodeLocally(tc.file, tc.title)
		if !ok || got != tc.want {
			t.Fatalf("%q / %q => %d, %v; want %d", tc.file, tc.title, got, ok, tc.want)
		}
	}
}
