package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const historyDir = ".chat_history"

type sessionFile struct {
	SavedAt  time.Time `json:"saved_at"`
	Messages []message `json:"messages"`
	Summary  string    `json:"summary,omitempty"`
}

func sessionPath(name string) string {
	if name == "" {
		name = "default"
	}
	return filepath.Join(historyDir, name+".json")
}

func saveSessionCW(name string, cw *contextWindow) error {
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(sessionFile{
		SavedAt:  time.Now().UTC(),
		Messages: cw.Messages,
		Summary:  cw.Summary,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sessionPath(name), data, 0644)
}

func loadSessionCW(name string) (*contextWindow, error) {
	data, err := os.ReadFile(sessionPath(name))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var sf sessionFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, err
	}
	return &contextWindow{
		Messages: sf.Messages,
		Summary:  sf.Summary,
	}, nil
}

func deleteSession(name string) error {
	err := os.Remove(sessionPath(name))
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
