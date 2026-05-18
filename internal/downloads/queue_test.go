package downloads

import "testing"

func TestPlanDownloadsQueuesOnlyUntriggeredItemsWithDownloadURL(t *testing.T) {
	items := []FeedItem{
		{ID: 1, DownloadURL: "https://example.com/one.torrent", DownloadStatus: StatusPending},
		{ID: 2, DownloadURL: "https://example.com/two.torrent", DownloadStatus: StatusSubmitted},
		{ID: 3, DownloadURL: "", DownloadStatus: StatusPending},
	}

	jobs := PlanDownloads(items)
	if len(jobs) != 1 {
		t.Fatalf("jobs = %d, want 1", len(jobs))
	}
	if jobs[0].ItemID != 1 || jobs[0].URL != "https://example.com/one.torrent" {
		t.Fatalf("job = %#v", jobs[0])
	}
}
