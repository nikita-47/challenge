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
	Strategy    string  `json:"strategy,omitempty"`    // "summary"|"window"|"facts"|"branch"
	WindowSize  int     `json:"window_size,omitempty"` // N for window/facts strategies (default 20)
	Profile     string  `json:"profile,omitempty"`     // memory profile name
	Project     string  `json:"project,omitempty"`     // memory project name
	Operator    string  `json:"operator,omitempty"`    // operator profile name
}

type sessionStats struct {
	TotalInput  int `json:"total_input"`
	TotalOutput int `json:"total_output"`
	Exchanges   int `json:"exchanges"`
	TokensSaved int `json:"tokens_saved"`
}

func statsFromToken(ts tokenStats) *sessionStats {
	return &sessionStats{
		TotalInput:  ts.TotalInput,
		TotalOutput: ts.TotalOutput,
		Exchanges:   ts.Exchanges,
		TokensSaved: ts.TokensSaved,
	}
}

func (ss *sessionStats) toTokenStats() tokenStats {
	if ss == nil {
		return tokenStats{}
	}
	return tokenStats{
		TotalInput:  ss.TotalInput,
		TotalOutput: ss.TotalOutput,
		Exchanges:   ss.Exchanges,
		TokensSaved: ss.TokensSaved,
	}
}

type sessionFile struct {
	SavedAt      time.Time         `json:"saved_at"`
	Messages     []message         `json:"messages"`
	Summary      string            `json:"summary,omitempty"`
	Settings     *sessionSettings  `json:"settings,omitempty"`
	Stats        *sessionStats     `json:"stats,omitempty"`
	Facts        map[string]string `json:"facts,omitempty"`
	Branches     []branch          `json:"branches,omitempty"`
	ActiveBranch string            `json:"active_branch,omitempty"`
	TaskState    *TaskState        `json:"task_state,omitempty"`
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
		SavedAt:      time.Now().UTC(),
		Messages:     cw.Messages,
		Summary:      cw.Summary,
		Settings:     cw.Settings,
		Stats:        cw.Stats,
		Facts:        cw.Facts,
		Branches:     cw.Branches,
		ActiveBranch: cw.ActiveBranch,
		TaskState:    cw.TaskState,
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
		Messages:     sf.Messages,
		Summary:      sf.Summary,
		Settings:     sf.Settings,
		Stats:        sf.Stats,
		Facts:        sf.Facts,
		Branches:     sf.Branches,
		ActiveBranch: sf.ActiveBranch,
		TaskState:    sf.TaskState,
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

type sessionInfo struct {
	Name     string `json:"name"`
	Profile  string `json:"profile,omitempty"`
	Project  string `json:"project,omitempty"`
	Operator string `json:"operator,omitempty"`
}

// listSessions returns info about all saved sessions.
func listSessions() ([]sessionInfo, error) {
	entries, err := os.ReadDir(historyDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var result []sessionInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		sName := strings.TrimSuffix(name, ".json")
		info := sessionInfo{Name: sName}
		if data, err := os.ReadFile(filepath.Join(historyDir, name)); err == nil {
			var sf sessionFile
			if json.Unmarshal(data, &sf) == nil && sf.Settings != nil {
				info.Profile = sf.Settings.Profile
				info.Project = sf.Settings.Project
				info.Operator = sf.Settings.Operator
			}
		}
		result = append(result, info)
	}
	return result, nil
}
