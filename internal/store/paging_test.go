package store

import "testing"

func TestNormalizePage(t *testing.T) {
	t.Parallel()
	page, size, offset := NormalizePage(0, 0)
	if page != 1 || size != DefaultPageSize || offset != 0 {
		t.Fatalf("got page=%d size=%d offset=%d", page, size, offset)
	}
	page, size, offset = NormalizePage(2, 200)
	if page != 2 || size != MaxPageSize || offset != MaxPageSize {
		t.Fatalf("got page=%d size=%d offset=%d", page, size, offset)
	}
	page, size, offset = NormalizePage(3, 10)
	if page != 3 || size != 10 || offset != 20 {
		t.Fatalf("got page=%d size=%d offset=%d", page, size, offset)
	}
}
