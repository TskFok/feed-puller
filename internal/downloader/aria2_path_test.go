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

func TestAria2DownloadPath_Missing(t *testing.T) {
	t.Parallel()
	_, err := Aria2DownloadPath(map[string]any{"status": "complete"})
	if err == nil {
		t.Fatal("expected error")
	}
}
