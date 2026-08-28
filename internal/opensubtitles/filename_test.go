package opensubtitles

import "testing"

func TestSanitizeFileName(t *testing.T) {
	t.Parallel()
	got, err := SanitizeFileName("/tmp/nested/foo.srt")
	if err != nil || got != "foo.srt" {
		t.Fatalf("got %q err=%v", got, err)
	}
	if _, err := SanitizeFileName(".."); err == nil {
		t.Fatal("expected error")
	}
	if _, err := SanitizeFileName(" . "); err == nil {
		t.Fatal("expected error")
	}
}

func TestSanitizeFileName_RejectsEmptyAndDot(t *testing.T) {
	t.Parallel()
	if _, err := SanitizeFileName(""); err == nil {
		t.Fatal("expected error")
	}
	if _, err := SanitizeFileName("   "); err == nil {
		t.Fatal("expected error")
	}
	if _, err := SanitizeFileName("."); err == nil {
		t.Fatal("expected error")
	}
}
