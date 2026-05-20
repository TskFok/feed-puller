package downloader

import "testing"

func TestIsAria2DownloadReady_OnlyMetadataNotReady(t *testing.T) {
	t.Parallel()
	ready := IsAria2DownloadReady(map[string]any{
		"status": "complete",
		"files": []any{
			map[string]any{
				"path":            "/data/[METADATA][ANi]+foo+.mp4",
				"completedLength": "100",
				"length":          "100",
			},
		},
	})
	if ready {
		t.Fatal("expected not ready when only metadata file is complete")
	}
}

func TestIsAria2DownloadReady_RealFileComplete(t *testing.T) {
	t.Parallel()
	ready := IsAria2DownloadReady(map[string]any{
		"status": "complete",
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
	if !ready {
		t.Fatal("expected ready when real media file is complete")
	}
}

func TestIsAria2DownloadReady_ActiveNotReady(t *testing.T) {
	t.Parallel()
	ready := IsAria2DownloadReady(map[string]any{
		"status": "active",
		"files": []any{
			map[string]any{
				"path":            "/data/[ANi]foo.mp4",
				"completedLength": "1",
				"length":          "1000",
			},
		},
	})
	if ready {
		t.Fatal("expected not ready when still active")
	}
}

func TestIsAria2DownloadReady_RealFileIncompleteDespiteCompleteStatus(t *testing.T) {
	t.Parallel()
	ready := IsAria2DownloadReady(map[string]any{
		"status": "complete",
		"files": []any{
			map[string]any{
				"path":            "/data/[METADATA][ANi]+foo+.mp4",
				"completedLength": "100",
				"length":          "100",
			},
			map[string]any{
				"path":            "/data/[ANi]foo - 07.mp4",
				"completedLength": "1",
				"length":          "1000",
			},
		},
	})
	if ready {
		t.Fatal("expected not ready when real file has length info but is incomplete")
	}
}
