package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_Valid(t *testing.T) {
	dir := t.TempDir()
	content := "[paths]\nsource = \"C:\\\\src\"\ndestination = \"C:\\\\dst\"\n\n[ignore]\npatterns = [\".stfolder\", \".data\"]"
	if err := os.WriteFile(filepath.Join(dir, "sync.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Paths.Source != `C:\src` {
		t.Errorf("source: got %q, want %q", cfg.Paths.Source, `C:\src`)
	}
	if cfg.Paths.Destination != `C:\dst` {
		t.Errorf("destination: got %q, want %q", cfg.Paths.Destination, `C:\dst`)
	}
	if len(cfg.Ignore.Patterns) != 2 {
		t.Errorf("patterns: got %d, want 2", len(cfg.Ignore.Patterns))
	}
}

func TestLoadConfig_Missing(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadConfig(dir)
	if err == nil {
		t.Error("expected error for missing sync.toml, got nil")
	}
}

func TestLoadConfig_EmptyPaths(t *testing.T) {
	dir := t.TempDir()
	content := "[paths]\nsource = \"\"\ndestination = \"\""
	if err := os.WriteFile(filepath.Join(dir, "sync.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(dir)
	if err == nil {
		t.Error("expected error for empty paths, got nil")
	}
}
