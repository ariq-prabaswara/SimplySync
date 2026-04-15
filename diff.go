package main

import (
	"path/filepath"
)

type OpKind int

const (
	OpCopy   OpKind = iota // file exists on one side only (new)
	OpUpdate               // file exists on both sides, one is newer
	OpDelete               // file was in snapshot but is now missing from one side
)

type Operation struct {
	Kind    OpKind
	RelPath string
	Src     string // for Copy/Update: absolute path to read from
	Dst     string // for Copy/Update: absolute path to write to; for Delete: path to remove
	ToSrc   bool   // true when copying/updating toward the source root
}

// ComputeOps compares src and dst file maps against the snapshot and returns
// the list of operations needed to bring both sides into sync.
func ComputeOps(srcFiles, dstFiles map[string]ScannedFile, snapshot *Snapshot, srcRoot, dstRoot string) []Operation {
	var ops []Operation
	seen := make(map[string]bool)

	for rel, sf := range srcFiles {
		seen[rel] = true
		df, inDst := dstFiles[rel]
		_, inSnap := snapshot.Files[rel]

		switch {
		case !inDst && !inSnap:
			// New file at source — copy to destination
			ops = append(ops, Operation{
				Kind:    OpCopy,
				RelPath: rel,
				Src:     sf.AbsPath,
				Dst:     filepath.ToSlash(filepath.Join(dstRoot, filepath.FromSlash(rel))),
				ToSrc:   false,
			})
		case !inDst && inSnap:
			// Was in snapshot, missing from dst — deleted at dst → delete at src
			ops = append(ops, Operation{
				Kind:    OpDelete,
				RelPath: rel,
				Dst:     sf.AbsPath,
			})
		case inDst && sf.ModTime.Equal(df.ModTime):
			// Unchanged — skip
		case inDst && sf.ModTime.After(df.ModTime):
			// Source newer — copy src → dst
			ops = append(ops, Operation{
				Kind:    OpUpdate,
				RelPath: rel,
				Src:     sf.AbsPath,
				Dst:     df.AbsPath,
				ToSrc:   false,
			})
		case inDst && df.ModTime.After(sf.ModTime):
			// Destination newer — copy dst → src
			ops = append(ops, Operation{
				Kind:    OpUpdate,
				RelPath: rel,
				Src:     df.AbsPath,
				Dst:     sf.AbsPath,
				ToSrc:   true,
			})
		}
	}

	for rel, df := range dstFiles {
		if seen[rel] {
			continue
		}
		_, inSnap := snapshot.Files[rel]
		if !inSnap {
			// New file at destination — copy to source
			ops = append(ops, Operation{
				Kind:    OpCopy,
				RelPath: rel,
				Src:     df.AbsPath,
				Dst:     filepath.ToSlash(filepath.Join(srcRoot, filepath.FromSlash(rel))),
				ToSrc:   true,
			})
		} else {
			// Was in snapshot, missing from src — deleted at src → delete at dst
			ops = append(ops, Operation{
				Kind:    OpDelete,
				RelPath: rel,
				Dst:     df.AbsPath,
			})
		}
	}

	return ops
}
