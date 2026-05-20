package rename

import "testing"

func TestFormatScrapeName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		season  int
		episode int
		ext     string
		want    string
	}{
		{"鬼灭之刃", 1, 1, ".mp4", "鬼灭之刃 S01E01.mp4"},
		{"Demon Slayer", 2, 12, ".mkv", "Demon Slayer S02E12.mkv"},
		{"test<>:file", 1, 1, ".mp4", "test file S01E01.mp4"},
	}
	for _, tc := range cases {
		got := FormatScrapeName(tc.name, tc.season, tc.episode, tc.ext)
		if got != tc.want {
			t.Fatalf("FormatScrapeName(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}
