package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const memoryDir = ".memory"

func memoryProfilesDir() string { return filepath.Join(memoryDir, "profiles") }
func memoryProjectsDir() string { return filepath.Join(memoryDir, "projects") }

// validateMemoryName ensures safe file names — no path traversal.
func validateMemoryName(name string) error {
	if name == "" {
		return errors.New("name is required")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return errors.New("invalid name")
	}
	if filepath.Base(name) != name {
		return errors.New("invalid name")
	}
	return nil
}

func listMemoryFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
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
		n := e.Name()
		if strings.HasSuffix(n, ".md") {
			names = append(names, strings.TrimSuffix(n, ".md"))
		}
	}
	return names, nil
}

func getMemoryFile(dir, name string) (string, error) {
	if err := validateMemoryName(name); err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(dir, name+".md"))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func saveMemoryFile(dir, name, content string) error {
	if err := validateMemoryName(name); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name+".md"), []byte(content), 0644)
}

func deleteMemoryFile(dir, name string) error {
	if err := validateMemoryName(name); err != nil {
		return err
	}
	err := os.Remove(filepath.Join(dir, name+".md"))
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// ─── Profile helpers ─────────────────────────────────────────────────────────

func listProfiles() ([]string, error)          { return listMemoryFiles(memoryProfilesDir()) }
func getProfile(name string) (string, error)   { return getMemoryFile(memoryProfilesDir(), name) }
func saveProfile(name, content string) error    { return saveMemoryFile(memoryProfilesDir(), name, content) }
func deleteProfile(name string) error           { return deleteMemoryFile(memoryProfilesDir(), name) }

// ─── Project helpers ─────────────────────────────────────────────────────────

func listProjects() ([]string, error)          { return listMemoryFiles(memoryProjectsDir()) }
func getProject(name string) (string, error)   { return getMemoryFile(memoryProjectsDir(), name) }
func saveProject(name, content string) error    { return saveMemoryFile(memoryProjectsDir(), name, content) }
func deleteProject(name string) error           { return deleteMemoryFile(memoryProjectsDir(), name) }
