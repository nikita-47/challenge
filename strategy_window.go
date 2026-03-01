package main

// buildWindowMessages returns only the last N messages (sliding window strategy).
func buildWindowMessages(cw *contextWindow) []message {
	n := getWindowSize(cw)
	msgs := filterAPIMessages(activeMessages(cw))
	if len(msgs) > n {
		msgs = msgs[len(msgs)-n:]
	}
	return msgs
}
