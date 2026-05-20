package rename

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureRenamePermissions_OwnerCanChmod(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "a.mp4")
	if err := os.WriteFile(from, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	if err := ensureRenamePermissions(from, filepath.Join(dir, "b.mp4")); err != nil {
		t.Fatalf("ensureRenamePermissions: %v", err)
	}
	if !dirIsWritable(dir) {
		t.Fatal("expected directory writable after chmod")
	}
}
