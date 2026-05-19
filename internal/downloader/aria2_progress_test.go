package downloader

import "testing"

func TestParseAria2Progress(t *testing.T) {
	t.Parallel()
	status := map[string]any{
		"status":          "active",
		"completedLength": "500",
		"totalLength":     "1000",
		"downloadSpeed":   "1024",
	}
	got := ParseAria2Progress(status)
	if got.Status != "active" {
		t.Fatalf("status = %q", got.Status)
	}
	if got.CompletedLength != 500 || got.TotalLength != 1000 || got.DownloadSpeed != 1024 {
		t.Fatalf("lengths/speed = %+v", got)
	}
	if got.ProgressPercent == nil || *got.ProgressPercent != 50 {
		t.Fatalf("percent = %v", got.ProgressPercent)
	}
}

func TestParseAria2ProgressNoTotal(t *testing.T) {
	t.Parallel()
	got := ParseAria2Progress(map[string]any{"status": "waiting"})
	if got.ProgressPercent != nil {
		t.Fatalf("expected nil percent, got %v", got.ProgressPercent)
	}
}
