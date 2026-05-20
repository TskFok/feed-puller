package rename

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestRenameFile(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "old.mp4")
	to := filepath.Join(dir, "new S01E01.mp4")
	if err := os.WriteFile(from, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RenameFile(from, to); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(to); err != nil {
		t.Fatalf("target missing: %v", err)
	}
	if _, err := os.Stat(from); !os.IsNotExist(err) {
		t.Fatal("source should be gone")
	}
}

func TestRenameFileSamePathNoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "same.mp4")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RenameFile(path, path); err != nil {
		t.Fatal(err)
	}
}

func TestShouldRenameByCopy(t *testing.T) {
	t.Parallel()
	if !shouldRenameByCopy(syscall.Errno(syscall.EXDEV)) {
		t.Fatal("expected EXDEV to trigger copy fallback")
	}
	if shouldRenameByCopy(syscall.Errno(syscall.EACCES)) {
		t.Fatal("EACCES should not trigger copy fallback")
	}
}

func TestIsPermissionError(t *testing.T) {
	t.Parallel()
	if !isPermissionError(syscall.Errno(syscall.EACCES)) {
		t.Fatal("expected EACCES")
	}
	if !isPermissionError(errors.Join(syscall.Errno(syscall.EPERM))) {
		t.Fatal("expected EPERM")
	}
}

func TestCopyFilePreservesMode(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.mp4")
	dst := filepath.Join(dir, "dst.mp4")
	if err := os.WriteFile(src, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
}
