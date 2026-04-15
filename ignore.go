package main

import (
	"path/filepath"
	"strings"
)

type IgnoreMatcher struct {
	patterns []string
}

func NewIgnoreMatcher(patterns []string) *IgnoreMatcher {
	return &IgnoreMatcher{patterns: patterns}
}

// ShouldIgnore returns true if any path component matches a pattern.
// relPath must use forward slashes or OS-native separators.
func (m *IgnoreMatcher) ShouldIgnore(relPath string) bool {
	relPath = filepath.ToSlash(relPath)
	parts := strings.Split(relPath, "/")
	for _, part := range parts {
		for _, pattern := range m.patterns {
			if part == pattern {
				return true
			}
		}
	}
	return false
}
