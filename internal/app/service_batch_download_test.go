package app

import "testing"

func TestSubmitItemDownloads_Empty(t *testing.T) {
	t.Parallel()
	s := &Service{}
	items, failures := s.SubmitItemDownloads(t.Context(), nil)
	if len(items) != 0 || len(failures) != 0 {
		t.Fatalf("expected empty result, got items=%d failures=%d", len(items), len(failures))
	}
}

func TestSubmitItemDownloads_TooMany(t *testing.T) {
	t.Parallel()
	s := &Service{}
	ids := make([]int64, maxBatchItemDownloads+1)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	_, failures := s.SubmitItemDownloads(t.Context(), ids)
	if len(failures) != 1 || failures[0].ItemID != 0 {
		t.Fatalf("expected single limit failure, got %+v", failures)
	}
}
