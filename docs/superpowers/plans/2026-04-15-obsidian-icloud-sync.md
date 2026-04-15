# ObsidianSync Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a compiled Windows `.exe` that two-way syncs an Obsidian vault with an iCloud Drive folder, driven by a `sync.toml` config file with ignore patterns and a JSON snapshot to track deletions.

**Architecture:** Single Go binary reads `sync.toml` for paths and ignore patterns, walks both directories, computes needed operations by comparing mtimes against a JSON snapshot, prompts the user if deletions are pending (case-insensitive Y/N), then executes all copies/deletes and updates the snapshot. Cancelling the prompt cancels all operations.

**Tech Stack:** Go 1.21+, `github.com/BurntSushi/toml`, Go standard library (`os`, `path/filepath`, `encoding/json`, `bufio`, `time`, `io`).

---

## File Structure

All source files live in `C:\Users\rifar\ObsidianSync\` with module name `obsync`.

| File | Responsibility |
|---|---|
| `go.mod` / `go.sum` | Module definition and dependencies |
| `config.go` | Load and validate `sync.toml` |
| `ignore.go` | Match file/dir path components against ignore patterns |
| `snapshot.go` | Load/save `sync-state.json` (tracks file state after last sync) |
| `scanner.go` | Walk a directory tree, build `map[relPath]ScannedFile` |
| `diff.go` | Compare scan results + snapshot → produce `[]Operation` |
| `ops.go` | Execute file copy (with mtime preservation) and delete |
| `logger.go` | Append timestamped entries to `sync.log` |
| `prompt.go` | Display pending deletions warning and read Y/N |
| `main.go` | Entry point — orchestrates all components |

Runtime-only files (not committed):
- `sync.toml` — user config
- `sync-state.json` — auto-managed snapshot
- `sync.log` — appended each run
- `obsync.exe` — compiled binary

---

### Task 1: Initialize Go module

**Files:**
- Create: `C:\Users\rifar\ObsidianSync\go.mod`

- [ ] **Step 1: Create project directory and initialize module**

```bash
mkdir -p "C:\Users\rifar\ObsidianSync"
cd "C:\Users\rifar\ObsidianSync"
go mod init obsync
```

Expected: `go.mod` created with `module obsync` and Go version line.

- [ ] **Step 2: Add TOML dependency**

```bash
go get github.com/BurntSushi/toml@latest
```

Expected: `go.sum` created, `go.mod` updated with a `require` entry for `github.com/BurntSushi/toml`.

- [ ] **Step 3: Initialize git and commit**

```bash
git init
git add go.mod go.sum
git commit -m "chore: initialize obsync Go module"
```

---

### Task 2: Config loading

**Files:**
- Create: `C:\Users\rifar\ObsidianSync\config.go`
- Create: `C:\Users\rifar\ObsidianSync\config_test.go`

- [ ] **Step 1: Write failing tests**

Create `config_test.go`:

```go
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
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
cd "C:\Users\rifar\ObsidianSync"
go test -run TestLoadConfig -v
```

Expected: FAIL — `LoadConfig undefined`

- [ ] **Step 3: Implement config.go**

Create `config.go`:

```go
package main

import (
	"fmt"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Paths  PathsConfig  `toml:"paths"`
	Ignore IgnoreConfig `toml:"ignore"`
}

type PathsConfig struct {
	Source      string `toml:"source"`
	Destination string `toml:"destination"`
}

type IgnoreConfig struct {
	Patterns []string `toml:"patterns"`
}

func LoadConfig(exeDir string) (*Config, error) {
	path := filepath.Join(exeDir, "sync.toml")
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("cannot load sync.toml: %w", err)
	}
	if cfg.Paths.Source == "" || cfg.Paths.Destination == "" {
		return nil, fmt.Errorf("sync.toml: [paths] source and destination must not be empty")
	}
	return &cfg, nil
}
```

- [ ] **Step 4: Run tests — verify they pass**

```bash
go test -run TestLoadConfig -v
```

Expected: PASS all three tests.

- [ ] **Step 5: Commit**

```bash
git add config.go config_test.go
git commit -m "feat: config loading from sync.toml"
```

---

### Task 3: Ignore pattern matching

**Files:**
- Create: `C:\Users\rifar\ObsidianSync\ignore.go`
- Create: `C:\Users\rifar\ObsidianSync\ignore_test.go`

- [ ] **Step 1: Write failing tests**

Create `ignore_test.go`:

```go
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
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
go test -run TestIgnoreMatcher -v
```

Expected: FAIL — `NewIgnoreMatcher undefined`

- [ ] **Step 3: Implement ignore.go**

Create `ignore.go`:

```go
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
```

- [ ] **Step 4: Run tests — verify they pass**

```bash
go test -run TestIgnoreMatcher -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add ignore.go ignore_test.go
git commit -m "feat: ignore pattern matching by path component"
```

---

### Task 4: Snapshot management

**Files:**
- Create: `C:\Users\rifar\ObsidianSync\snapshot.go`
- Create: `C:\Users\rifar\ObsidianSync\snapshot_test.go`

- [ ] **Step 1: Write failing tests**

Create `snapshot_test.go`:

```go
package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSnapshot_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sync-state.json")
	now := time.Now().UTC().Truncate(time.Second)

	snap := &Snapshot{
		Files: map[string]time.Time{
			"notes/daily.md": now,
			"inbox/idea.md":  now.Add(-time.Hour),
		},
	}
	if err := snap.Save(path); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	loaded, err := LoadSnapshot(path)
	if err != nil {
		t.Fatalf("LoadSnapshot error: %v", err)
	}
	if len(loaded.Files) != 2 {
		t.Errorf("got %d files, want 2", len(loaded.Files))
	}
	if !loaded.Files["notes/daily.md"].Equal(now) {
		t.Errorf("mtime mismatch for notes/daily.md: got %v, want %v", loaded.Files["notes/daily.md"], now)
	}
}

func TestSnapshot_MissingFile(t *testing.T) {
	dir := t.TempDir()
	snap, err := LoadSnapshot(filepath.Join(dir, "nonexistent.json"))
	if err != nil {
		t.Fatalf("unexpected error for missing snapshot: %v", err)
	}
	if snap.Files == nil {
		t.Error("expected non-nil Files map")
	}
	if len(snap.Files) != 0 {
		t.Errorf("expected empty map, got %d entries", len(snap.Files))
	}
}
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
go test -run TestSnapshot -v
```

Expected: FAIL — `Snapshot undefined`

- [ ] **Step 3: Implement snapshot.go**

Create `snapshot.go`:

```go
package main

import (
	"encoding/json"
	"errors"
	"os"
	"time"
)

// Snapshot records which files existed on both sides after the last sync,
// and their modification times. Used to detect deletions.
type Snapshot struct {
	Files map[string]time.Time `json:"files"`
}

// LoadSnapshot reads sync-state.json. If the file does not exist, returns
// an empty snapshot (first run — no deletions propagated).
func LoadSnapshot(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Snapshot{Files: map[string]time.Time{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	if snap.Files == nil {
		snap.Files = map[string]time.Time{}
	}
	return &snap, nil
}

// Save writes the snapshot to path as indented JSON.
func (s *Snapshot) Save(path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
```

- [ ] **Step 4: Run tests — verify they pass**

```bash
go test -run TestSnapshot -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add snapshot.go snapshot_test.go
git commit -m "feat: snapshot load/save for deletion tracking"
```

---

### Task 5: Directory scanner

**Files:**
- Create: `C:\Users\rifar\ObsidianSync\scanner.go`
- Create: `C:\Users\rifar\ObsidianSync\scanner_test.go`

- [ ] **Step 1: Write failing tests**

Create `scanner_test.go`:

```go
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
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
go test -run TestScanDir -v
```

Expected: FAIL — `ScanDir undefined`

- [ ] **Step 3: Implement scanner.go**

Create `scanner.go`:

```go
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
```

- [ ] **Step 4: Run tests — verify they pass**

```bash
go test -run TestScanDir -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add scanner.go scanner_test.go
git commit -m "feat: directory scanner with ignore pattern support"
```

---

### Task 6: Diff computation

**Files:**
- Create: `C:\Users\rifar\ObsidianSync\diff.go`
- Create: `C:\Users\rifar\ObsidianSync\diff_test.go`

- [ ] **Step 1: Write failing tests**

Create `diff_test.go`:

```go
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
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
go test -run TestComputeOps -v
```

Expected: FAIL — `ComputeOps undefined`

- [ ] **Step 3: Implement diff.go**

Create `diff.go`:

```go
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
				Dst:     filepath.Join(dstRoot, filepath.FromSlash(rel)),
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
				Dst:     filepath.Join(srcRoot, filepath.FromSlash(rel)),
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
```

- [ ] **Step 4: Run tests — verify they pass**

```bash
go test -run TestComputeOps -v
```

Expected: PASS all seven tests.

- [ ] **Step 5: Commit**

```bash
git add diff.go diff_test.go
git commit -m "feat: two-way diff computation with snapshot-based deletion detection"
```

---

### Task 7: File operations

**Files:**
- Create: `C:\Users\rifar\ObsidianSync\ops.go`
- Create: `C:\Users\rifar\ObsidianSync\ops_test.go`

- [ ] **Step 1: Write failing tests**

Create `ops_test.go`:

```go
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
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
go test -run "TestCopyFile|TestDeleteFile" -v
```

Expected: FAIL — `CopyFile undefined`

- [ ] **Step 3: Implement ops.go**

Create `ops.go`:

```go
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
```

- [ ] **Step 4: Run tests — verify they pass**

```bash
go test -run "TestCopyFile|TestDeleteFile" -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add ops.go ops_test.go
git commit -m "feat: file copy with mtime preservation and delete"
```

---

### Task 8: Logger

**Files:**
- Create: `C:\Users\rifar\ObsidianSync\logger.go`
- Create: `C:\Users\rifar\ObsidianSync\logger_test.go`

- [ ] **Step 1: Write failing tests**

Create `logger_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogger_WritesToFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "sync.log")
	l, err := NewLogger(logPath)
	if err != nil {
		t.Fatalf("NewLogger error: %v", err)
	}
	l.Log("test message %d", 42)
	l.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if !strings.Contains(string(data), "test message 42") {
		t.Errorf("log missing expected content, got: %s", string(data))
	}
}

func TestLogger_AppendsAcrossRuns(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "sync.log")

	l1, _ := NewLogger(logPath)
	l1.Log("first run")
	l1.Close()

	l2, _ := NewLogger(logPath)
	l2.Log("second run")
	l2.Close()

	data, _ := os.ReadFile(logPath)
	content := string(data)
	if !strings.Contains(content, "first run") || !strings.Contains(content, "second run") {
		t.Errorf("expected both log entries, got: %s", content)
	}
}

func TestLogger_NilFileSafe(t *testing.T) {
	l := &Logger{}
	// Must not panic with nil file
	l.Log("no-op")
	l.Close()
}
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
go test -run TestLogger -v
```

Expected: FAIL — `NewLogger undefined`

- [ ] **Step 3: Implement logger.go**

Create `logger.go`:

```go
package main

import (
	"fmt"
	"os"
	"time"
)

type Logger struct {
	file *os.File
}

// NewLogger opens (or creates) logPath in append mode.
func NewLogger(logPath string) (*Logger, error) {
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{file: f}, nil
}

// Log writes a timestamped line. Safe to call on a zero-value Logger.
func (l *Logger) Log(format string, args ...interface{}) {
	if l.file == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	ts := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(l.file, "[%s] %s\n", ts, msg)
}

// Close closes the underlying file. Safe to call on a zero-value Logger.
func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}
```

- [ ] **Step 4: Run tests — verify they pass**

```bash
go test -run TestLogger -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add logger.go logger_test.go
git commit -m "feat: append-mode file logger with timestamps"
```

---

### Task 9: Confirmation prompt

**Files:**
- Create: `C:\Users\rifar\ObsidianSync\prompt.go`
- Create: `C:\Users\rifar\ObsidianSync\prompt_test.go`

- [ ] **Step 1: Write failing tests**

Create `prompt_test.go`:

```go
package main

import (
	"strings"
	"testing"
)

var mixedOps = []Operation{
	{Kind: OpCopy,   RelPath: "inbox/new.md",  Src: "/src/inbox/new.md",  Dst: "/dst/inbox/new.md"},
	{Kind: OpUpdate, RelPath: "notes/daily.md", Src: "/src/notes/daily.md", Dst: "/dst/notes/daily.md"},
	{Kind: OpDelete, RelPath: "old/gone.md",    Dst: "/dst/old/gone.md"},
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
		{Kind: OpCopy,   RelPath: "new.md",  Src: "/src/new.md",  Dst: "/dst/new.md"},
		{Kind: OpUpdate, RelPath: "note.md", Src: "/src/note.md", Dst: "/dst/note.md"},
	}
	// Empty reader — if it prompts, scanner.Scan() returns false → would return false.
	// Correct behavior: return true without reading.
	if !confirmSync(noDeletions, strings.NewReader("")) {
		t.Error("expected true when no deletions pending")
	}
}
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
go test -run TestConfirmSync -v
```

Expected: FAIL — `confirmSync undefined`

- [ ] **Step 3: Implement prompt.go**

Create `prompt.go`:

```go
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// ConfirmSync is the public entry point used by main. Reads from os.Stdin.
func ConfirmSync(ops []Operation) bool {
	return confirmSync(ops, os.Stdin)
}

// confirmSync shows a deletion warning and reads Y/N from r.
// Returns true immediately (no read) when there are no pending deletions.
// Input is case-insensitive; anything other than y/Y is treated as cancel.
func confirmSync(ops []Operation, r io.Reader) bool {
	var deletions []Operation
	var copies, updates int
	for _, op := range ops {
		switch op.Kind {
		case OpDelete:
			deletions = append(deletions, op)
		case OpCopy:
			copies++
		case OpUpdate:
			updates++
		}
	}

	if len(deletions) == 0 {
		return true
	}

	fmt.Printf("\n⚠ Warning: %d file(s) will be deleted:\n", len(deletions))
	for _, d := range deletions {
		fmt.Printf("    %s\n", d.RelPath)
	}
	fmt.Printf("\nProceed with sync? (includes %d copies, %d updates, %d deletions) [Y/N]: ", copies, updates, len(deletions))

	scanner := bufio.NewScanner(r)
	if scanner.Scan() {
		if strings.ToLower(strings.TrimSpace(scanner.Text())) == "y" {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run tests — verify they pass**

```bash
go test -run TestConfirmSync -v
```

Expected: PASS all five tests.

- [ ] **Step 5: Commit**

```bash
git add prompt.go prompt_test.go
git commit -m "feat: case-insensitive Y/N deletion confirmation prompt"
```

---

### Task 10: Main orchestration

**Files:**
- Create: `C:\Users\rifar\ObsidianSync\main.go`

- [ ] **Step 1: Verify full test suite passes before writing main**

```bash
cd "C:\Users\rifar\ObsidianSync"
go test ./... -v
```

Expected: All tests PASS.

- [ ] **Step 2: Implement main.go**

Create `main.go`:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const version = "1.0"

func main() {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine executable path: %v\n", err)
		waitAndExit(1)
	}
	exeDir := filepath.Dir(exePath)

	cfg, err := LoadConfig(exeDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		waitAndExit(1)
	}

	fmt.Printf("ObsidianSync v%s\n", version)
	fmt.Printf("Source:      %s\n", cfg.Paths.Source)
	fmt.Printf("Destination: %s\n\n", cfg.Paths.Destination)

	logPath := filepath.Join(exeDir, "sync.log")
	logger, err := NewLogger(logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cannot open log file: %v\n", err)
		logger = &Logger{}
	} else {
		defer logger.Close()
	}
	logger.Log("Sync started")

	snapPath := filepath.Join(exeDir, "sync-state.json")
	snapshot, err := LoadSnapshot(snapPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading snapshot: %v\n", err)
		waitAndExit(1)
	}

	fmt.Println("Scanning...")
	matcher := NewIgnoreMatcher(cfg.Ignore.Patterns)

	srcFiles, err := ScanDir(cfg.Paths.Source, matcher)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning source: %v\n", err)
		waitAndExit(1)
	}
	dstFiles, err := ScanDir(cfg.Paths.Destination, matcher)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning destination: %v\n", err)
		waitAndExit(1)
	}

	ops := ComputeOps(srcFiles, dstFiles, snapshot, cfg.Paths.Source, cfg.Paths.Destination)

	if len(ops) == 0 {
		fmt.Println("\nNothing to sync. Everything is up to date.")
		logger.Log("Sync complete. 0 operations.")
		waitAndExit(0)
	}

	if !ConfirmSync(ops) {
		fmt.Println("Sync cancelled. No changes made.")
		logger.Log("Sync cancelled by user.")
		waitAndExit(0)
	}

	fmt.Println()

	// Build the new snapshot starting from the current state of srcFiles.
	// We update it as operations complete.
	newSnap := &Snapshot{Files: make(map[string]time.Time)}
	for rel, sf := range srcFiles {
		newSnap.Files[rel] = sf.ModTime
	}

	var copied, updated, deleted, failed int

	for _, op := range ops {
		switch op.Kind {
		case OpCopy, OpUpdate:
			label := "[→]"
			if op.ToSrc {
				label = "[←]"
			}
			fmt.Printf("%s %s\n", label, op.RelPath)
			if err := CopyFile(op.Src, op.Dst); err != nil {
				fmt.Printf("    ERROR: %v\n", err)
				logger.Log("ERROR %s %s: %v", label, op.RelPath, err)
				failed++
			} else {
				srcInfo, statErr := os.Stat(op.Src)
				if statErr == nil {
					newSnap.Files[op.RelPath] = srcInfo.ModTime().UTC().Truncate(time.Second)
				}
				if op.Kind == OpCopy {
					copied++
					logger.Log("Copied: %s (%s)", op.RelPath, label)
				} else {
					updated++
					logger.Log("Updated: %s (%s)", op.RelPath, label)
				}
			}
		case OpDelete:
			fmt.Printf("[✕] %s\n", op.RelPath)
			if err := DeleteFile(op.Dst); err != nil {
				fmt.Printf("    ERROR: %v\n", err)
				logger.Log("ERROR deleting %s: %v", op.RelPath, err)
				failed++
			} else {
				delete(newSnap.Files, op.RelPath)
				deleted++
				logger.Log("Deleted: %s", op.RelPath)
			}
		}
	}

	unchanged := len(srcFiles) - copied - updated - deleted - failed
	if unchanged < 0 {
		unchanged = 0
	}

	summary := fmt.Sprintf("\nDone. %d copied, %d updated, %d deleted, %d unchanged", copied, updated, deleted, unchanged)
	if failed > 0 {
		summary += fmt.Sprintf(", %d errors", failed)
	}
	summary += "."
	fmt.Println(summary)
	logger.Log("Sync complete. %d copied, %d updated, %d deleted, %d errors.", copied, updated, deleted, failed)

	if err := newSnap.Save(snapPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save snapshot: %v\n", err)
	}

	waitAndExit(0)
}

func waitAndExit(code int) {
	fmt.Println("\nPress any key to exit...")
	var b [1]byte
	os.Stdin.Read(b[:]) //nolint:errcheck
	os.Exit(code)
}
```

- [ ] **Step 3: Verify project builds without errors**

```bash
cd "C:\Users\rifar\ObsidianSync"
go build ./...
```

Expected: No errors. Do NOT run the resulting binary.

- [ ] **Step 4: Run full test suite**

```bash
go test ./... -v
```

Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add main.go
git commit -m "feat: main orchestration — wires all sync components"
```

---

### Task 11: Build binary, config file, and desktop shortcut

**Files:**
- Create: `C:\Users\rifar\ObsidianSync\obsync.exe` (compiled output)
- Create: `C:\Users\rifar\ObsidianSync\sync.toml`
- Create: `C:\Users\rifar\Desktop\ObsidianSync.lnk`

- [ ] **Step 1: Compile the binary**

```bash
cd "C:\Users\rifar\ObsidianSync"
go build -o obsync.exe .
```

Expected: `obsync.exe` appears in `C:\Users\rifar\ObsidianSync\`. Do NOT run it.

- [ ] **Step 2: Create sync.toml**

Create `C:\Users\rifar\ObsidianSync\sync.toml` with this content:

```toml
[paths]
source      = 'C:\Ariq\Jade Chamber\Obsidian'
destination = 'C:\Users\rifar\iCloudDrive\iCloud~md~obsidian\Jade Chamber'

[ignore]
patterns = [
    ".stfolder",
    ".stignore",
    ".stversions",
    ".data",
]
```

- [ ] **Step 3: Create desktop shortcut via PowerShell**

```bash
powershell -Command "$ws = New-Object -ComObject WScript.Shell; $s = $ws.CreateShortcut('C:\Users\rifar\Desktop\ObsidianSync.lnk'); $s.TargetPath = 'C:\Users\rifar\ObsidianSync\obsync.exe'; $s.WorkingDirectory = 'C:\Users\rifar\ObsidianSync'; $s.Save()"
```

Expected: `ObsidianSync.lnk` appears on the desktop.

- [ ] **Step 4: Commit**

```bash
git add sync.toml
git commit -m "chore: add sync.toml config; build obsync.exe"
```
