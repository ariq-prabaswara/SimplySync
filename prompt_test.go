package main

import (
	"strings"
	"testing"
)

var mixedOps = []Operation{
	{Kind: OpCopy, RelPath: "inbox/new.md", Src: "/src/inbox/new.md", Dst: "/dst/inbox/new.md"},
	{Kind: OpUpdate, RelPath: "notes/daily.md", Src: "/src/notes/daily.md", Dst: "/dst/notes/daily.md"},
	{Kind: OpDelete, RelPath: "old/gone.md", Dst: "/dst/old/gone.md"},
}

func TestConfirmSync_YLower(t *testing.T) {
	if !confirmSync(mixedOps, strings.NewReader("y\n")) {
		t.Error("expected true for 'y'")
	}
}

func TestConfirmSync_YUpper(t *testing.T) {
	if !confirmSync(mixedOps, strings.NewReader("Y\n")) {
		t.Error("expected true for 'Y'")
	}
}

func TestConfirmSync_NLower(t *testing.T) {
	if confirmSync(mixedOps, strings.NewReader("n\n")) {
		t.Error("expected false for 'n'")
	}
}

func TestConfirmSync_NUpper(t *testing.T) {
	if confirmSync(mixedOps, strings.NewReader("N\n")) {
		t.Error("expected false for 'N'")
	}
}

func TestConfirmSync_NoDeletions_NoPrompt(t *testing.T) {
	noDeletions := []Operation{
		{Kind: OpCopy, RelPath: "new.md", Src: "/src/new.md", Dst: "/dst/new.md"},
		{Kind: OpUpdate, RelPath: "note.md", Src: "/src/note.md", Dst: "/dst/note.md"},
	}
	// Empty reader — if it prompts, scanner.Scan() returns false → would return false.
	// Correct behavior: return true without reading.
	if !confirmSync(noDeletions, strings.NewReader("")) {
		t.Error("expected true when no deletions pending")
	}
}
