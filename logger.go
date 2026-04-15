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
