package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"sort"
	"sync"
	"time"
)

// TaskType represents the kind of scheduled task.
type TaskType string

const (
	TypeReminder   TaskType = "reminder"
	TypeURLMonitor TaskType = "url_monitor"
	TypeHNDigest   TaskType = "hn_digest"
	TypePipeline   TaskType = "pipeline"
)

// TaskStatus represents the current state of a task.
type TaskStatus string

const (
	StatusActive TaskStatus = "active"
	StatusPaused TaskStatus = "paused"
	StatusDone   TaskStatus = "done"
)

// TaskResult stores the outcome of a single task execution.
type TaskResult struct {
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
	Data      string    `json:"data"`
	Error     string    `json:"error,omitempty"`
}

// Task is a scheduled unit of work.
type Task struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      TaskType          `json:"type"`
	Status    TaskStatus        `json:"status"`
	Interval  string            `json:"interval"`
	Params    map[string]string `json:"params"`
	CreatedAt time.Time         `json:"created_at"`
	NextRun   time.Time         `json:"next_run"`
	Results   []TaskResult      `json:"results"`
}

const maxResults = 50

// Store manages tasks with thread-safe access and JSON persistence.
type Store struct {
	mu        sync.RWMutex
	tasks     map[string]*Task
	filePath  string
	runners   map[string]context.CancelFunc
	dirty     bool
	saveTimer *time.Timer
}

// NewStore creates a new Store backed by the given file path.
func NewStore(filePath string) *Store {
	return &Store{
		tasks:    make(map[string]*Task),
		filePath: filePath,
		runners:  make(map[string]context.CancelFunc),
	}
}

// Load reads tasks from the JSON file into memory.
func (s *Store) Load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var tasks []*Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range tasks {
		s.tasks[t.ID] = t
	}
	return nil
}

// Save writes all tasks to the JSON file.
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		tasks = append(tasks, t)
	}

	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}

	s.dirty = false
	return os.WriteFile(s.filePath, data, 0644)
}

// debouncedSave marks dirty and schedules a save after 2 seconds.
// Must be called with the write lock held.
func (s *Store) debouncedSave() {
	s.dirty = true
	if s.saveTimer != nil {
		s.saveTimer.Stop()
	}
	s.saveTimer = time.AfterFunc(2*time.Second, func() {
		_ = s.Save()
	})
}

// Add inserts a task into the store and persists.
func (s *Store) Add(task *Task) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks[task.ID] = task
	s.debouncedSave()
}

// Get retrieves a task by ID.
func (s *Store) Get(id string) (*Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tasks[id]
	return t, ok
}

// List returns all tasks sorted by creation time (ascending).
func (s *Store) List() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		tasks = append(tasks, t)
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
	return tasks
}

// Delete removes a task by ID, cancels its runner, and persists.
// Returns false if the task was not found.
func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; !ok {
		return false
	}

	if cancel, ok := s.runners[id]; ok {
		cancel()
		delete(s.runners, id)
	}

	delete(s.tasks, id)
	s.debouncedSave()
	return true
}

// UpdateStatus changes the status of a task and persists.
func (s *Store) UpdateStatus(id string, status TaskStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if t, ok := s.tasks[id]; ok {
		t.Status = status
		s.debouncedSave()
	}
}

// AppendResult adds a result to the task's ring buffer (max 50) and debounce-saves.
func (s *Store) AppendResult(id string, result TaskResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return
	}

	t.Results = append(t.Results, result)
	if len(t.Results) > maxResults {
		t.Results = t.Results[len(t.Results)-maxResults:]
	}
	s.debouncedSave()
}

// SetRunner stores the cancel function for a running task.
func (s *Store) SetRunner(id string, cancel context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runners[id] = cancel
}

// CancelRunner cancels and removes the runner for a task.
func (s *Store) CancelRunner(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cancel, ok := s.runners[id]; ok {
		cancel()
		delete(s.runners, id)
	}
}

// generateID returns a random 8-character hex string.
func generateID() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
