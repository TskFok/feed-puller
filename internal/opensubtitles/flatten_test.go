package opensubtitles

import "testing"

func TestFlattenSearchData_SkipsInvalidFileIDAndSplitsFiles(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"data":[{"attributes":{"language":"zh-CN","download_count":10,"ratings":8.5,"release":"Show.S01","feature_details":{"movie_name":"Show","title":"Ep"},"files":[{"file_id":101,"file_name":"a.srt"},{"file_id":102,"file_name":"b.ass"}]}},{"attributes":{"language":"en","files":[{"file_id":0,"file_name":"skip.srt"}]}}]}`)
	items := FlattenSearchData(raw)
	if len(items) != 2 || items[0].FileID != 101 || items[1].FileName != "b.ass" || items[0].Release != "Show.S01" {
		t.Fatalf("items=%+v", items)
	}
}

func TestFlattenSearchData_ReleaseFallbackMovieNameThenTitle(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"data":[{"attributes":{"feature_details":{"movie_name":"Movie","title":"Title"},"files":[{"file_id":1,"file_name":"a.srt"}]}},{"attributes":{"feature_details":{"title":"OnlyTitle"},"files":[{"file_id":2,"file_name":"b.srt"}]}}]}`)
	items := FlattenSearchData(raw)
	if len(items) != 2 || items[0].Release != "Movie" || items[1].Release != "OnlyTitle" {
		t.Fatalf("items=%+v", items)
	}
}

func TestFlattenSearchData_InvalidJSON(t *testing.T) {
	t.Parallel()
	if items := FlattenSearchData([]byte(`not json`)); len(items) != 0 {
		t.Fatalf("items=%+v", items)
	}
}

func TestFlattenSearchData_SortsByDownloadCountDesc(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"data":[{"attributes":{"download_count":3,"files":[{"file_id":1,"file_name":"low.srt"}]}},{"attributes":{"download_count":30,"files":[{"file_id":2,"file_name":"high.srt"},{"file_id":3,"file_name":"high-2.srt"}]}},{"attributes":{"download_count":10,"files":[{"file_id":4,"file_name":"mid.srt"}]}}]}`)
	items := FlattenSearchData(raw)
	if len(items) != 4 {
		t.Fatalf("len=%d items=%+v", len(items), items)
	}
	got := []int{items[0].DownloadCount, items[1].DownloadCount, items[2].DownloadCount, items[3].DownloadCount}
	want := []int{30, 30, 10, 3}
	if got[0] != want[0] || got[1] != want[1] || got[2] != want[2] || got[3] != want[3] {
		t.Fatalf("download_count=%v want %v items=%+v", got, want, items)
	}
	if items[0].FileID != 2 || items[1].FileID != 3 {
		t.Fatalf("equal download_count should keep flatten order, items=%+v", items)
	}
}

func TestParseSearchResponse_ReadsPaginationAndFlattens(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"page":2,"total_pages":5,"total_count":234,"data":[{"attributes":{"release":"Show","language":"zh-CN","download_count":1,"files":[{"file_id":101,"file_name":"a.srt"}]}}]}`)
	got := ParseSearchResponse(raw, 1)
	if got.Page != 2 || got.TotalPages != 5 || got.TotalCount != 234 {
		t.Fatalf("meta=%+v", got)
	}
	if len(got.Items) != 1 || got.Items[0].FileID != 101 || got.Items[0].FileName != "a.srt" {
		t.Fatalf("items=%+v", got.Items)
	}
}

func TestParseSearchResponse_MissingMetaUsesRequestPageAndZeroTotals(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"data":[{"attributes":{"files":[{"file_id":1,"file_name":"a.srt"}]}}]}`)
	got := ParseSearchResponse(raw, 3)
	if got.Page != 3 || got.TotalPages != 0 || got.TotalCount != 0 {
		t.Fatalf("meta=%+v", got)
	}
	if len(got.Items) != 1 || got.Items[0].FileID != 1 {
		t.Fatalf("items=%+v", got.Items)
	}
}

func TestParseSearchResponse_InvalidJSON(t *testing.T) {
	t.Parallel()
	got := ParseSearchResponse([]byte(`not json`), 2)
	if got.Page != 2 || got.TotalPages != 0 || got.TotalCount != 0 || len(got.Items) != 0 {
		t.Fatalf("got=%+v", got)
	}
}
