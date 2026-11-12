package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFailedReplacementPreservesDestination(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.txt")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(path, "keep.txt")
	if err := os.WriteFile(marker, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, []byte("replacement")); err == nil {
		t.Fatal("expected replacement of a directory to fail")
	}
	data, err := os.ReadFile(marker)
	if err != nil || string(data) != "original" {
		t.Fatalf("destination was changed: %q, %v", data, err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".netsentry-*.tmp"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary files leaked: %v, %v", matches, err)
	}
}
