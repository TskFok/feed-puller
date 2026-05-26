package rename

import "testing"

func TestBuildMovieTargetPath(t *testing.T) {
	t.Parallel()
	target, err := BuildMovieTargetPath("/data/Inception.2010.mkv", "Inception", 2010)
	if err != nil {
		t.Fatal(err)
	}
	if target != "/data/Inception (2010).mkv" {
		t.Fatalf("got %q", target)
	}
}

func TestBuildTVTargetPath(t *testing.T) {
	t.Parallel()
	target, err := BuildTVTargetPath("/data/ep.mkv", "Demo Show", 1, 5)
	if err != nil {
		t.Fatal(err)
	}
	if target != "/data/Demo Show - S01E05.mkv" {
		t.Fatalf("got %q", target)
	}
}
