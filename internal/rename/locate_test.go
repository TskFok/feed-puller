package rename

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindLargestMediaFileInDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "[METADATA]x.mp4"), []byte("tiny"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "small.mp4"), []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "large.mkv"), make([]byte, 200), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := FindLargestMediaFileInDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "large.mkv" {
		t.Fatalf("got %q, want large.mkv", got)
	}
}
