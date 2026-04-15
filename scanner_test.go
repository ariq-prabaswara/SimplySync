package main

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0755)
		os.WriteFile(p, []byte(content), 0644)
	}
	write("notes/daily.md", "hello")
	write("inbox/idea.md", "world")
	write(".stfolder/config", "ignored")
	write(".stignore", "ignored")
	write(".data/cache.db", "ignored")
	return root
}

func TestScanDir_IncludesNormalFiles(t *testing.T) {
	root := setupTestDir(t)
	matcher := NewIgnoreMatcher([]string{".stfolder", ".stignore", ".stversions", ".data"})
	files, err := ScanDir(root, matcher)
	if err != nil {
		t.Fatalf("ScanDir error: %v", err)
	}
	for _, want := range []string{"notes/daily.md", "inbox/idea.md"} {
		if _, ok := files[want]; !ok {
			t.Errorf("expected %q in results", want)
		}
	}
}

func TestScanDir_ExcludesIgnoredPaths(t *testing.T) {
	root := setupTestDir(t)
	matcher := NewIgnoreMatcher([]string{".stfolder", ".stignore", ".stversions", ".data"})
	files, err := ScanDir(root, matcher)
	if err != nil {
		t.Fatalf("ScanDir error: %v", err)
	}
	for _, bad := range []string{".stfolder/config", ".stignore", ".data/cache.db"} {
		if _, ok := files[bad]; ok {
			t.Errorf("expected %q to be ignored but it was included", bad)
		}
	}
}

func TestScanDir_ModTimePopulated(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "note.md"), []byte("content"), 0644)

	matcher := NewIgnoreMatcher(nil)
	files, err := ScanDir(root, matcher)
	if err != nil {
		t.Fatalf("ScanDir error: %v", err)
	}
	f, ok := files["note.md"]
	if !ok {
		t.Fatal("expected note.md in results")
	}
	if f.ModTime.IsZero() {
		t.Error("expected non-zero ModTime")
	}
	if f.AbsPath == "" {
		t.Error("expected non-empty AbsPath")
	}
}
