package main

import "fmt"

type tokenUsage struct {
	InputTokens  int
	OutputTokens int
}

type tokenStats struct {
	TotalInput  int
	TotalOutput int
	Exchanges   int
	TokensSaved int // estimated input tokens saved by compression
}

func (s *tokenStats) Add(u tokenUsage) {
	s.TotalInput += u.InputTokens
	s.TotalOutput += u.OutputTokens
	s.Exchanges++
}

func (s *tokenStats) TotalTokens() int {
	return s.TotalInput + s.TotalOutput
}

// TotalCost returns cost in USD. Claude Sonnet 4.5: $3/1M input, $15/1M output.
func (s *tokenStats) TotalCost() float64 {
	return float64(s.TotalInput)*3.0/1e6 + float64(s.TotalOutput)*15.0/1e6
}

func formatTokenUsage(u tokenUsage) string {
	return fmt.Sprintf("\033[2m[tokens: %d in / %d out]\033[0m", u.InputTokens, u.OutputTokens)
}

func (s *tokenStats) FormatTotal() string {
	base := fmt.Sprintf("\033[2m[session: %d in / %d out | %d exchanges | $%.6f",
		s.TotalInput, s.TotalOutput, s.Exchanges, s.TotalCost())
	if s.TokensSaved > 0 {
		base += fmt.Sprintf(" | saved ~%d tokens", s.TokensSaved)
	}
	return base + "]\033[0m"
}
