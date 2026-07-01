package app

import "testing"

func TestCanSubmitItemDownload(t *testing.T) {
	t.Parallel()
	allowed := []string{"pending", "preview", "failed", "submitted", "skipped", ""}
	for _, status := range allowed {
		if !CanSubmitItemDownload(status) {
			t.Fatalf("status %q should be allowed", status)
		}
	}
	if CanSubmitItemDownload("submitting") {
		t.Fatal("submitting should not be allowed")
	}
}
