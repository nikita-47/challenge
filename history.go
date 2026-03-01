package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const historyDir = ".chat_history"

type sessionSettings struct {
	Model       string  `json:"model,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	System      string  `json:"system,omitempty"`
}

type sessionFile struct {
	SavedAt  time.Time        `json:"saved_at"`
	Messages []message        `json:"messages"`
	Summary  string           `json:"summary,omitempty"`
	Settings *sessionSettings `json:"settings,omitempty"`
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
		Settings: cw.Settings,
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
		Settings: sf.Settings,
	}, nil
}

func deleteSession(name string) error {
	err := os.Remove(sessionPath(name))
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func renameSession(oldName, newName string) error {
	newPath := sessionPath(newName)
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("session %q already exists", newName)
	}
	return os.Rename(sessionPath(oldName), newPath)
}

// listSessions returns names of all saved sessions.
func listSessions() ([]string, error) {
	entries, err := os.ReadDir(historyDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".json") {
			names = append(names, strings.TrimSuffix(name, ".json"))
		}
	}
	return names, nil
}
