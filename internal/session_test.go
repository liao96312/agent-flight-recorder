package afr

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func TestSessionIDMillionUnique(t *testing.T) {
	seen := make(map[string]struct{}, 1_000_000)
	pattern := regexp.MustCompile(`^[0-9]{8}T[0-9]{6}\.[0-9]{9}Z-[0-9a-f]{24}$`)
	now := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	for range 1_000_000 {
		id, err := newSessionID(now)
		if err != nil {
			t.Fatal(err)
		}
		if !pattern.MatchString(id) {
			t.Fatalf("invalid id: %q", id)
		}
		if _, exists := seen[id]; exists {
			t.Fatalf("collision: %s", id)
		}
		seen[id] = struct{}{}
	}
}

func TestAtomicWriteFailureLeavesNoTemporaryFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "occupied")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteJSON(root, "occupied", map[string]string{"x": "y"}, testRedactor(t)); err == nil {
		t.Fatal("write over directory succeeded")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "occupied" || !entries[0].IsDir() {
		t.Fatalf("failure left partial files: %v", entries)
	}
}
