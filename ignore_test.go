package main

import "testing"

func TestIgnoreMatcher(t *testing.T) {
	m := NewIgnoreMatcher([]string{".stfolder", ".stignore", ".stversions", ".data"})
	cases := []struct {
		path string
		want bool
	}{
		{".stignore", true},
		{"subdir/.stignore", true},
		{".stfolder/somefile.txt", true},
		{".stversions/old.md", true},
		{".data/cache.db", true},
		{"notes/daily.md", false},
		{"00-Inbox/idea.md", false},
		{".obsidian/app.json", false},
		{"deep/nested/.stfolder/x", true},
	}
	for _, tc := range cases {
		got := m.ShouldIgnore(tc.path)
		if got != tc.want {
			t.Errorf("ShouldIgnore(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestIgnoreMatcher_EmptyPatterns(t *testing.T) {
	m := NewIgnoreMatcher(nil)
	if m.ShouldIgnore("anything.md") {
		t.Error("expected false with no patterns")
	}
}
