package downloader

import "testing"

func TestAria2DownloadPath(t *testing.T) {
	t.Parallel()
	got, err := Aria2DownloadPath(map[string]any{
		"files": []any{
			map[string]any{"path": "/data/anime/xxx 02.mp4"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "/data/anime/xxx 02.mp4" {
		t.Fatalf("path = %q", got)
	}
}

func TestAria2DownloadPath_SkipsMetadata(t *testing.T) {
	t.Parallel()
	got, err := Aria2DownloadPath(map[string]any{
		"files": []any{
			map[string]any{
				"path":            "/data/[METADATA][ANi]+foo+.mp4",
				"completedLength": "100",
				"length":          "100",
			},
			map[string]any{
				"path":            "/data/[ANi]foo - 07.mp4",
				"completedLength": "1000",
				"length":          "1000",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "/data/[ANi]foo - 07.mp4" {
		t.Fatalf("path = %q", got)
	}
}

func TestAria2DownloadPath_Missing(t *testing.T) {
	t.Parallel()
	_, err := Aria2DownloadPath(map[string]any{"status": "complete"})
	if err == nil {
		t.Fatal("expected error")
	}
}
