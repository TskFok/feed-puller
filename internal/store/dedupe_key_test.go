package store

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestProwlarrDedupeKey_ShortGUIDUnchanged(t *testing.T) {
	t.Parallel()
	got := ProwlarrDedupeKey(" g1 ")
	if got != "prowlarr:g1" {
		t.Fatalf("ProwlarrDedupeKey() = %q, want %q", got, "prowlarr:g1")
	}
}

func TestProwlarrDedupeKey_LongGUIDFitsColumnAndIsStable(t *testing.T) {
	t.Parallel()
	guid := strings.Repeat("a", MaxDedupeKeyLength)
	raw := "prowlarr:" + guid
	if utf8.RuneCountInString(raw) <= MaxDedupeKeyLength {
		t.Fatalf("precondition failed: raw key length = %d", utf8.RuneCountInString(raw))
	}

	got := ProwlarrDedupeKey(guid)
	if utf8.RuneCountInString(got) > MaxDedupeKeyLength {
		t.Fatalf("hashed key length = %d, want <= %d", utf8.RuneCountInString(got), MaxDedupeKeyLength)
	}
	if got == raw {
		t.Fatal("expected oversized key to be hashed")
	}
	if got != ProwlarrDedupeKey(guid) {
		t.Fatal("hashed key must be deterministic")
	}

	sum := sha256.Sum256([]byte(raw))
	want := "sha256:" + hex.EncodeToString(sum[:])
	if got != want {
		t.Fatalf("ProwlarrDedupeKey() = %q, want %q", got, want)
	}
}

func TestNormalizeDedupeKey_BoundaryLengthUnchanged(t *testing.T) {
	t.Parallel()
	key := strings.Repeat("x", MaxDedupeKeyLength)
	if got := NormalizeDedupeKey(key); got != key {
		t.Fatalf("NormalizeDedupeKey() = %q, want original 768-rune key", got)
	}
}

func TestNormalizeDedupeKey_DifferentInputsHashDifferently(t *testing.T) {
	t.Parallel()
	a := NormalizeDedupeKey(strings.Repeat("a", MaxDedupeKeyLength+1))
	b := NormalizeDedupeKey(strings.Repeat("b", MaxDedupeKeyLength+1))
	if a == b {
		t.Fatal("different oversized keys hashed to the same value")
	}
}

func TestMigrationsDedupeKeyMatchesMaxLength(t *testing.T) {
	t.Parallel()
	want := fmt.Sprintf("dedupe_key VARCHAR(%d)", MaxDedupeKeyLength)
	for _, stmt := range migrations {
		if strings.Contains(stmt, want) {
			return
		}
	}
	t.Fatalf("migrations must define %s to match MaxDedupeKeyLength", want)
}
