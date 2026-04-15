package main

import (
	"os"
	"path/filepath"
	"time"
)

type ScannedFile struct {
	RelPath string
	AbsPath string
	ModTime time.Time
}

// ScanDir walks root and returns a map of forward-slash relPath → ScannedFile.
// Directories and files matching the ignore matcher are skipped entirely.
func ScanDir(root string, matcher *IgnoreMatcher) (map[string]ScannedFile, error) {
	files := make(map[string]ScannedFile)
	err := filepath.WalkDir(root, func(absPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, absPath)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if matcher.ShouldIgnore(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		files[rel] = ScannedFile{
			RelPath: rel,
			AbsPath: absPath,
			ModTime: info.ModTime().UTC().Truncate(time.Second),
		}
		return nil
	})
	return files, err
}
