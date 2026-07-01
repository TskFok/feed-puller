package app

import "testing"

func TestCanSubmitItemDownload_AllowsPreview(t *testing.T) {
	t.Parallel()
	if !CanSubmitItemDownload("preview") {
		t.Fatal("preview items should be manually downloadable")
	}
}
