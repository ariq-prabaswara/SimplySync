package main

import (
	"io"
	"os"
	"path/filepath"
)

// CopyFile copies src to dst, creating parent directories as needed.
// Preserves the source file's modification time on the destination.
func CopyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime())
}

// DeleteFile removes the file at path.
func DeleteFile(path string) error {
	return os.Remove(path)
}
