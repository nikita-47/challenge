package main

const (
	strategySummary = "summary"
	strategyWindow  = "window"
	strategyFacts   = "facts"
	strategyBranch  = "branch"
)

// getStrategy returns the active strategy name, defaulting to "summary".
func getStrategy(cw *contextWindow) string {
	if cw.Settings != nil && cw.Settings.Strategy != "" {
		return cw.Settings.Strategy
	}
	return strategySummary
}

// buildAPIMessages dispatches to the correct strategy to produce the message
// slice sent to the Claude API.
func buildAPIMessages(cw *contextWindow) []message {
	switch getStrategy(cw) {
	case strategyWindow:
		return buildWindowMessages(cw)
	case strategyFacts:
		return buildFactsMessages(cw)
	case strategyBranch:
		return buildBranchMessages(cw)
	default:
		return buildCompressedMessages(cw)
	}
}

// maybeProcess dispatches pre-message processing per strategy.
// Called BEFORE appending the new user message so the current question isn't swallowed.
func maybeProcess(apiKey string, cw *contextWindow, stats *tokenStats) (*compressInfo, error) {
	switch getStrategy(cw) {
	case strategyWindow:
		return nil, nil // no pre-processing needed
	case strategyFacts:
		return nil, nil // facts extracted AFTER assistant response
	case strategyBranch:
		return nil, nil // branching has no auto-compression
	default:
		return maybeCompress(apiKey, cw, stats)
	}
}

// activeMessages returns messages for the active branch (or cw.Messages for main).
func activeMessages(cw *contextWindow) []message {
	if cw.ActiveBranch == "" || cw.ActiveBranch == "main" {
		return cw.Messages
	}
	for _, b := range cw.Branches {
		if b.Name == cw.ActiveBranch {
			prefix := cw.Messages
			if b.ForkIndex < len(prefix) {
				prefix = prefix[:b.ForkIndex]
			}
			result := make([]message, 0, len(prefix)+1+len(b.Messages))
			result = append(result, prefix...)
			// Insert fork marker so the UI can show where branching started.
			result = append(result, message{
				Role:    "system",
				Content: b.Name,
				Event: &messageEvent{
					Type: "branch_fork",
				},
			})
			result = append(result, b.Messages...)
			return result
		}
	}
	return cw.Messages
}

// appendMessage adds a message to the active branch (or cw.Messages for main).
func appendMessage(cw *contextWindow, msg message) {
	if cw.ActiveBranch == "" || cw.ActiveBranch == "main" {
		cw.Messages = append(cw.Messages, msg)
		return
	}
	for i := range cw.Branches {
		if cw.Branches[i].Name == cw.ActiveBranch {
			cw.Branches[i].Messages = append(cw.Branches[i].Messages, msg)
			return
		}
	}
	cw.Messages = append(cw.Messages, msg)
}

// getWindowSize returns the window size from settings, defaulting to 20.
func getWindowSize(cw *contextWindow) int {
	if cw.Settings != nil && cw.Settings.WindowSize > 0 {
		return cw.Settings.WindowSize
	}
	return 20
}
