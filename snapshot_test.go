package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSnapshot_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sync-state.json")
	now := time.Now().UTC().Truncate(time.Second)

	snap := &Snapshot{
		Files: map[string]time.Time{
			"notes/daily.md": now,
			"inbox/idea.md":  now.Add(-time.Hour),
		},
	}
	if err := snap.Save(path); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	loaded, err := LoadSnapshot(path)
	if err != nil {
		t.Fatalf("LoadSnapshot error: %v", err)
	}
	if len(loaded.Files) != 2 {
		t.Errorf("got %d files, want 2", len(loaded.Files))
	}
	if !loaded.Files["notes/daily.md"].Equal(now) {
		t.Errorf("mtime mismatch for notes/daily.md: got %v, want %v", loaded.Files["notes/daily.md"], now)
	}
}

func TestSnapshot_MissingFile(t *testing.T) {
	dir := t.TempDir()
	snap, err := LoadSnapshot(filepath.Join(dir, "nonexistent.json"))
	if err != nil {
		t.Fatalf("unexpected error for missing snapshot: %v", err)
	}
	if snap.Files == nil {
		t.Error("expected non-nil Files map")
	}
	if len(snap.Files) != 0 {
		t.Errorf("expected empty map, got %d entries", len(snap.Files))
	}
}
