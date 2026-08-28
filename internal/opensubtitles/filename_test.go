package opensubtitles

import (
	"errors"
	"testing"
)

func TestSanitizeFileName(t *testing.T) {
	t.Parallel()
	got, err := SanitizeFileName("/tmp/nested/foo.srt")
	if err != nil || got != "foo.srt" {
		t.Fatalf("got %q err=%v", got, err)
	}
	if _, err := SanitizeFileName(".."); !errors.Is(err, ErrInvalidFileName) {
		t.Fatalf("err=%v", err)
	}
	if _, err := SanitizeFileName(" . "); !errors.Is(err, ErrInvalidFileName) {
		t.Fatalf("err=%v", err)
	}
}

func TestSanitizeFileName_RejectsEmptyAndDot(t *testing.T) {
	t.Parallel()
	if _, err := SanitizeFileName(""); !errors.Is(err, ErrInvalidFileName) {
		t.Fatalf("err=%v", err)
	}
	if _, err := SanitizeFileName("   "); !errors.Is(err, ErrInvalidFileName) {
		t.Fatalf("err=%v", err)
	}
	if _, err := SanitizeFileName("."); !errors.Is(err, ErrInvalidFileName) {
		t.Fatalf("err=%v", err)
	}
}
