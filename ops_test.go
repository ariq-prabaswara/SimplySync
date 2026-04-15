package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile_CreatesDestination(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.md")
	dst := filepath.Join(dir, "subdir", "dst.md")
	os.WriteFile(src, []byte("hello"), 0644)

	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("content: got %q, want %q", string(data), "hello")
	}
}

func TestCopyFile_PreservesModTime(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.md")
	dst := filepath.Join(dir, "dst.md")
	os.WriteFile(src, []byte("content"), 0644)
	srcInfo, _ := os.Stat(src)

	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}
	dstInfo, _ := os.Stat(dst)
	if !dstInfo.ModTime().Equal(srcInfo.ModTime()) {
		t.Errorf("ModTime not preserved: src=%v dst=%v", srcInfo.ModTime(), dstInfo.ModTime())
	}
}

func TestDeleteFile_RemovesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todelete.md")
	os.WriteFile(path, []byte("bye"), 0644)

	if err := DeleteFile(path); err != nil {
		t.Fatalf("DeleteFile error: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected file to be gone after DeleteFile")
	}
}
