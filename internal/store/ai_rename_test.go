package store

import "testing"

func TestNormalizeAIRenameSeason(t *testing.T) {
	t.Parallel()
	if got := normalizeAIRenameSeason(0); got != 1 {
		t.Fatalf("got %d, want 1", got)
	}
	if got := normalizeAIRenameSeason(3); got != 3 {
		t.Fatalf("got %d, want 3", got)
	}
}
