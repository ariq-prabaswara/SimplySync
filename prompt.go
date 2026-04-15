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
