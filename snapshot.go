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
