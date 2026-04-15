package main

import (
	"testing"
	"time"
)

func sf(rel, root string, mtime time.Time) ScannedFile {
	return ScannedFile{RelPath: rel, AbsPath: root + "/" + rel, ModTime: mtime}
}

var (
	t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 = t0.Add(time.Hour)
)

func TestComputeOps_NewAtSrc(t *testing.T) {
	src := map[string]ScannedFile{"new.md": sf("new.md", "/src", t0)}
	dst := map[string]ScannedFile{}
	snap := &Snapshot{Files: map[string]time.Time{}}
	ops := ComputeOps(src, dst, snap, "/src", "/dst")
	if len(ops) != 1 || ops[0].Kind != OpCopy || ops[0].ToSrc {
		t.Errorf("expected one OpCopy src→dst, got %+v", ops)
	}
	if ops[0].Src != "/src/new.md" || ops[0].Dst != "/dst/new.md" {
		t.Errorf("wrong paths: %+v", ops[0])
	}
}

func TestComputeOps_NewAtDst(t *testing.T) {
	src := map[string]ScannedFile{}
	dst := map[string]ScannedFile{"new.md": sf("new.md", "/dst", t0)}
	snap := &Snapshot{Files: map[string]time.Time{}}
	ops := ComputeOps(src, dst, snap, "/src", "/dst")
	if len(ops) != 1 || ops[0].Kind != OpCopy || !ops[0].ToSrc {
		t.Errorf("expected one OpCopy dst→src, got %+v", ops)
	}
	if ops[0].Src != "/dst/new.md" || ops[0].Dst != "/src/new.md" {
		t.Errorf("wrong paths: %+v", ops[0])
	}
}

func TestComputeOps_DeletedFromDst(t *testing.T) {
	// File was in snapshot but is now missing from dst → was deleted at dst → delete at src
	src := map[string]ScannedFile{"gone.md": sf("gone.md", "/src", t0)}
	dst := map[string]ScannedFile{}
	snap := &Snapshot{Files: map[string]time.Time{"gone.md": t0}}
	ops := ComputeOps(src, dst, snap, "/src", "/dst")
	if len(ops) != 1 || ops[0].Kind != OpDelete {
		t.Errorf("expected one OpDelete, got %+v", ops)
	}
	if ops[0].Dst != "/src/gone.md" {
		t.Errorf("expected delete at src path, got %q", ops[0].Dst)
	}
}

func TestComputeOps_DeletedFromSrc(t *testing.T) {
	// File was in snapshot but is now missing from src → was deleted at src → delete at dst
	src := map[string]ScannedFile{}
	dst := map[string]ScannedFile{"gone.md": sf("gone.md", "/dst", t0)}
	snap := &Snapshot{Files: map[string]time.Time{"gone.md": t0}}
	ops := ComputeOps(src, dst, snap, "/src", "/dst")
	if len(ops) != 1 || ops[0].Kind != OpDelete {
		t.Errorf("expected one OpDelete, got %+v", ops)
	}
	if ops[0].Dst != "/dst/gone.md" {
		t.Errorf("expected delete at dst path, got %q", ops[0].Dst)
	}
}

func TestComputeOps_Unchanged(t *testing.T) {
	src := map[string]ScannedFile{"same.md": sf("same.md", "/src", t0)}
	dst := map[string]ScannedFile{"same.md": sf("same.md", "/dst", t0)}
	snap := &Snapshot{Files: map[string]time.Time{"same.md": t0}}
	ops := ComputeOps(src, dst, snap, "/src", "/dst")
	if len(ops) != 0 {
		t.Errorf("expected no ops for unchanged file, got %+v", ops)
	}
}

func TestComputeOps_SrcNewer(t *testing.T) {
	src := map[string]ScannedFile{"note.md": sf("note.md", "/src", t1)}
	dst := map[string]ScannedFile{"note.md": sf("note.md", "/dst", t0)}
	snap := &Snapshot{Files: map[string]time.Time{"note.md": t0}}
	ops := ComputeOps(src, dst, snap, "/src", "/dst")
	if len(ops) != 1 || ops[0].Kind != OpUpdate || ops[0].ToSrc {
		t.Errorf("expected one OpUpdate src→dst, got %+v", ops)
	}
	if ops[0].Src != "/src/note.md" || ops[0].Dst != "/dst/note.md" {
		t.Errorf("wrong paths: %+v", ops[0])
	}
}

func TestComputeOps_DstNewer(t *testing.T) {
	src := map[string]ScannedFile{"note.md": sf("note.md", "/src", t0)}
	dst := map[string]ScannedFile{"note.md": sf("note.md", "/dst", t1)}
	snap := &Snapshot{Files: map[string]time.Time{"note.md": t0}}
	ops := ComputeOps(src, dst, snap, "/src", "/dst")
	if len(ops) != 1 || ops[0].Kind != OpUpdate || !ops[0].ToSrc {
		t.Errorf("expected one OpUpdate dst→src, got %+v", ops)
	}
	if ops[0].Src != "/dst/note.md" || ops[0].Dst != "/src/note.md" {
		t.Errorf("wrong paths: %+v", ops[0])
	}
}
