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
