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

	fmt.Printf("SimplySync v%s\n", version)
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
