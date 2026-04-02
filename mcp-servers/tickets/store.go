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

type TicketStatus string

const (
	StatusOpen   TicketStatus = "open"
	StatusClosed TicketStatus = "closed"
)

type TicketPriority string

const (
	PriorityLow    TicketPriority = "low"
	PriorityMedium TicketPriority = "medium"
	PriorityHigh   TicketPriority = "high"
)

type TicketMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type Ticket struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      TicketStatus    `json:"status"`
	Priority    TicketPriority  `json:"priority"`
	UserEmail   string          `json:"user_email,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	ClosedAt    *time.Time      `json:"closed_at,omitempty"`
	Resolution  string          `json:"resolution,omitempty"`
	Messages    []TicketMessage `json:"messages"`
}

type Store struct {
	mu        sync.RWMutex
	tickets   map[string]*Ticket
	filePath  string
	dirty     bool
	saveTimer *time.Timer
}

func NewStore(filePath string) *Store {
	return &Store{
		tickets:  make(map[string]*Ticket),
		filePath: filePath,
	}
}

func (s *Store) Load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var tickets []*Ticket
	if err := json.Unmarshal(data, &tickets); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range tickets {
		s.tickets[t.ID] = t
	}
	return nil
}

func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tickets := make([]*Ticket, 0, len(s.tickets))
	for _, t := range s.tickets {
		tickets = append(tickets, t)
	}

	data, err := json.MarshalIndent(tickets, "", "  ")
	if err != nil {
		return err
	}

	s.dirty = false
	return os.WriteFile(s.filePath, data, 0644)
}

func (s *Store) debouncedSave() {
	s.dirty = true
	if s.saveTimer != nil {
		s.saveTimer.Stop()
	}
	s.saveTimer = time.AfterFunc(2*time.Second, func() {
		_ = s.Save()
	})
}

func (s *Store) Add(ticket *Ticket) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tickets[ticket.ID] = ticket
	s.debouncedSave()
}

func (s *Store) Get(id string) (*Ticket, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tickets[id]
	return t, ok
}

func (s *Store) List() []*Ticket {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tickets := make([]*Ticket, 0, len(s.tickets))
	for _, t := range s.tickets {
		tickets = append(tickets, t)
	}
	sort.Slice(tickets, func(i, j int) bool {
		return tickets[i].CreatedAt.Before(tickets[j].CreatedAt)
	})
	return tickets
}

func (s *Store) Close(id, resolution string, closedAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if t, ok := s.tickets[id]; ok {
		t.Status = StatusClosed
		t.Resolution = resolution
		t.ClosedAt = &closedAt
		s.debouncedSave()
	}
}

func (s *Store) AddMessage(id string, msg TicketMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if t, ok := s.tickets[id]; ok {
		t.Messages = append(t.Messages, msg)
		s.debouncedSave()
	}
}

func generateID() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
