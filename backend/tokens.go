package main

import "fmt"

type tokenUsage struct {
	InputTokens        int `json:"input"`
	OutputTokens       int `json:"output"`
	CacheCreationInput int `json:"cache_creation_input,omitempty"`
	CacheReadInput     int `json:"cache_read_input,omitempty"`
}

type tokenStats struct {
	TotalInput         int
	TotalOutput        int
	Exchanges          int
	TokensSaved        int // estimated input tokens saved by compression
	CacheCreationInput int
	CacheReadInput     int
}

func (s *tokenStats) Add(u tokenUsage) {
	s.TotalInput += u.InputTokens
	s.TotalOutput += u.OutputTokens
	s.CacheCreationInput += u.CacheCreationInput
	s.CacheReadInput += u.CacheReadInput
	s.Exchanges++
}

func (s *tokenStats) TotalTokens() int {
	return s.TotalInput + s.TotalOutput
}

// TotalCost returns cost in USD.
// Claude Sonnet 4.5 pricing:
// Input: $3/M, Output: $15/M
// Cache write: $3.75/M, Cache read: $0.30/M
func (s *tokenStats) TotalCost() float64 {
	return float64(s.TotalInput)*3.0/1e6 +
		float64(s.TotalOutput)*15.0/1e6 +
		float64(s.CacheCreationInput)*3.75/1e6 +
		float64(s.CacheReadInput)*0.30/1e6
}

func formatTokenUsage(u tokenUsage) string {
	s := fmt.Sprintf("\033[2m[tokens: %d in / %d out", u.InputTokens, u.OutputTokens)
	if u.CacheReadInput > 0 || u.CacheCreationInput > 0 {
		s += fmt.Sprintf(" | cache: %d read, %d write", u.CacheReadInput, u.CacheCreationInput)
	}
	return s + "]\033[0m"
}

func (s *tokenStats) FormatTotal() string {
	base := fmt.Sprintf("\033[2m[session: %d in / %d out | %d exchanges | $%.6f",
		s.TotalInput, s.TotalOutput, s.Exchanges, s.TotalCost())
	if s.TokensSaved > 0 {
		base += fmt.Sprintf(" | saved ~%d tokens", s.TokensSaved)
	}
	if s.CacheCreationInput > 0 || s.CacheReadInput > 0 {
		base += fmt.Sprintf(" | cache: %d read, %d write", s.CacheReadInput, s.CacheCreationInput)
	}
	return base + "]\033[0m"
}
