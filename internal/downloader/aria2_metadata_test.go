package downloader

import "testing"

func TestIsMetadataDownloadPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		path string
		want bool
	}{
		{path: "/data/[METADATA][ANi]+foo+.mp4", want: true},
		{path: "/data/[ANi]大賢者 - 07.mp4", want: false},
		{path: "", want: false},
	}
	for _, tc := range cases {
		if got := IsMetadataDownloadPath(tc.path); got != tc.want {
			t.Fatalf("IsMetadataDownloadPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}
