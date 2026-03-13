package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"sort"
	"sync"
	"time"
)

// StepStatus represents the state of a single pipeline step.
type StepStatus string

const (
	StatusPending StepStatus = "pending"
	StatusRunning StepStatus = "running"
	StatusDone    StepStatus = "done"
	StatusError   StepStatus = "error"
)

// PipelineStep tracks the execution of one step within a pipeline run.
type PipelineStep struct {
	Name       string     `json:"name"`
	Status     StepStatus `json:"status"`
	StartedAt  time.Time  `json:"started_at,omitempty"`
	FinishedAt time.Time  `json:"finished_at,omitempty"`
	Output     string     `json:"output,omitempty"`
	Error      string     `json:"error,omitempty"`
}

// PipelineRun represents a full pipeline execution with all its steps.
type PipelineRun struct {
	ID         string         `json:"id"`
	Query      string         `json:"query"`
	Source     string         `json:"source"`
	Count      int            `json:"count"`
	Status     StepStatus     `json:"status"`
	Steps      []PipelineStep `json:"steps"`
	OutputFile string         `json:"output_file,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// Store manages pipeline runs with thread-safe access and JSON persistence.
type Store struct {
	mu        sync.RWMutex
	runs      map[string]*PipelineRun
	filePath  string
	dirty     bool
	saveTimer *time.Timer
}

// NewStore creates a new Store backed by the given file path.
func NewStore(filePath string) *Store {
	return &Store{
		runs:     make(map[string]*PipelineRun),
		filePath: filePath,
	}
}

// Load reads pipeline runs from the JSON file into memory.
func (s *Store) Load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var runs []*PipelineRun
	if err := json.Unmarshal(data, &runs); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, r := range runs {
		s.runs[r.ID] = r
	}
	return nil
}

// Save writes all runs to the JSON file.
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	runs := make([]*PipelineRun, 0, len(s.runs))
	for _, r := range s.runs {
		runs = append(runs, r)
	}

	data, err := json.MarshalIndent(runs, "", "  ")
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

// Add inserts a pipeline run into the store and persists.
func (s *Store) Add(run *PipelineRun) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runs[run.ID] = run
	s.debouncedSave()
}

// Get retrieves a pipeline run by ID.
func (s *Store) Get(id string) (*PipelineRun, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.runs[id]
	return r, ok
}

// List returns all runs sorted by creation time (ascending).
func (s *Store) List() []*PipelineRun {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runs := make([]*PipelineRun, 0, len(s.runs))
	for _, r := range s.runs {
		runs = append(runs, r)
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].CreatedAt.Before(runs[j].CreatedAt)
	})
	return runs
}

// Latest returns the most recently created run, or nil if there are none.
func (s *Store) Latest() *PipelineRun {
	runs := s.List()
	if len(runs) == 0 {
		return nil
	}
	return runs[len(runs)-1]
}

// UpdateStep updates the fields of a named step within a run and persists.
func (s *Store) UpdateStep(id string, stepName string, update func(*PipelineStep)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	run, ok := s.runs[id]
	if !ok {
		return
	}

	for i := range run.Steps {
		if run.Steps[i].Name == stepName {
			update(&run.Steps[i])
			break
		}
	}
	s.debouncedSave()
}

// UpdateStatus sets the overall status of a run and persists.
func (s *Store) UpdateStatus(id string, status StepStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if r, ok := s.runs[id]; ok {
		r.Status = status
		s.debouncedSave()
	}
}

// SetOutputFile records the output file path on a run and persists.
func (s *Store) SetOutputFile(id string, path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if r, ok := s.runs[id]; ok {
		r.OutputFile = path
		s.debouncedSave()
	}
}

// Delete removes a pipeline run by ID and persists.
func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.runs[id]; !ok {
		return false
	}

	delete(s.runs, id)
	s.debouncedSave()
	return true
}

// generateID returns a random 8-character hex string.
func generateID() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
