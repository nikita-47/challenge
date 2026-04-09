package main

import (
	"strings"
	"unicode"
)

// SanitizeInput removes leading/trailing whitespace and control characters.
func SanitizeInput(s string) string {
	return strings.TrimSpace(s)
}

// TruncateString cuts a string to maxLen and appends "..." if truncated.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// CountWords returns the number of words in a string.
func CountWords(s string) int {
	count := 0
	inWord := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			inWord = false
		} else if !inWord {
			inWord = true
			count++
		}
	}
	return count
}

// ContainsAny checks if the string contains any of the given substrings.
func ContainsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
