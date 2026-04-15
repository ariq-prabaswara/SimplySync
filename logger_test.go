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
