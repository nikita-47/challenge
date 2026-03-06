package main

import "fmt"

type tokenUsage struct {
	InputTokens        int `json:"input"`
	OutputTokens       int `json:"output"`
	CacheCreationInput int `json:"cache_creation_input,omitempty"`
	CacheReadInput     int `json:"cache_read_input,omitempty"`
}

type tokenStats struct {
	Model              string // model ID for pricing lookup
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

// TotalCost returns cost in USD using pricing for s.Model.
func (s *tokenStats) TotalCost() float64 {
	p := PricingFor(s.Model)
	return float64(s.TotalInput)*p.CostIn/1e6 +
		float64(s.TotalOutput)*p.CostOut/1e6 +
		float64(s.CacheCreationInput)*p.CacheWriteIn/1e6 +
		float64(s.CacheReadInput)*p.CacheReadIn/1e6
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
