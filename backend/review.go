package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

type pullRequest struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	Author string   `json:"author"`
	Branch string   `json:"branch"`
	URL    string   `json:"url"`
	Labels []string `json:"labels"`
}

type reviewRequest struct {
	PRNumber int `json:"pr_number"`
}

// listOpenPRs fetches open pull requests from GitHub using the gh CLI.
func listOpenPRs() ([]pullRequest, error) {
	cmd := exec.Command(
		"gh", "pr", "list",
		"--repo", "nikita-47/challenge",
		"--json", "number,title,author,headRefName,url,labels",
		"--state", "open",
	)

	out, err := cmd.Output()
	if err != nil {
		// Provide a clear error for missing gh or auth failure.
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh pr list failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh not found or failed: %w", err)
	}

	// gh outputs author as {"login":"..."} and labels as [{"name":"..."}].
	var raw []struct {
		Number      int    `json:"number"`
		Title       string `json:"title"`
		Author      struct {
			Login string `json:"login"`
		} `json:"author"`
		HeadRefName string `json:"headRefName"`
		URL         string `json:"url"`
		Labels      []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}

	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parse gh output: %w", err)
	}

	prs := make([]pullRequest, 0, len(raw))
	for _, r := range raw {
		labels := make([]string, 0, len(r.Labels))
		for _, l := range r.Labels {
			labels = append(labels, l.Name)
		}
		prs = append(prs, pullRequest{
			Number: r.Number,
			Title:  r.Title,
			Author: r.Author.Login,
			Branch: r.HeadRefName,
			URL:    r.URL,
			Labels: labels,
		})
	}

	return prs, nil
}

// getPRDiff fetches the git diff for a pull request. Truncates at 100KB.
func getPRDiff(prNumber int) (string, error) {
	cmd := exec.Command(
		"gh", "pr", "diff",
		"--repo", "nikita-47/challenge",
		strconv.Itoa(prNumber),
	)

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("gh pr diff failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("gh pr diff failed: %w", err)
	}

	const maxBytes = 100 * 1024
	if len(out) > maxBytes {
		return string(out[:maxBytes]) + "\n\n[DIFF TRUNCATED: exceeded 100KB limit]", nil
	}

	return string(out), nil
}

// getPRFiles returns the list of changed file names for a pull request.
func getPRFiles(prNumber int) ([]string, error) {
	cmd := exec.Command(
		"gh", "pr", "diff",
		"--repo", "nikita-47/challenge",
		"--name-only",
		strconv.Itoa(prNumber),
	)

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh pr diff --name-only failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh pr diff --name-only failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	files := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			files = append(files, l)
		}
	}

	return files, nil
}

// postPRComment posts a comment to a pull request via gh CLI.
func postPRComment(prNumber int, body string) error {
	cmd := exec.Command(
		"gh", "pr", "comment",
		"--repo", "nikita-47/challenge",
		strconv.Itoa(prNumber),
		"--body", body,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh pr comment failed: %s", string(out))
	}

	return nil
}

// buildReviewRAGContext attempts to enrich the review with project documentation context.
// Returns an XML string if relevant chunks found, or empty string if Ollama is unavailable.
func buildReviewRAGContext(apiKey string, docStore *DocumentStore, diff string) string {
	// Use the first 500 chars of the diff as a summary query.
	query := diff
	if len(query) > 500 {
		query = query[:500]
	}

	embedding, err := GetEmbedding(query)
	if err != nil {
		// Ollama down — skip RAG enrichment silently.
		return ""
	}

	docIDs := docStore.AllReadyDocIDs()
	if len(docIDs) == 0 {
		return ""
	}

	var allResults []SearchResult
	docNameByChunkID := make(map[string]string)

	for _, docID := range docIDs {
		doc := docStore.Get(docID)
		if doc == nil {
			continue
		}

		idx, loadErr := loadCombinedIndex(docStore.indexDir, docID)
		if loadErr != nil {
			continue
		}

		results := SearchChunks(idx, embedding, 5, "auto")
		for _, r := range results {
			allResults = append(allResults, r)
			docNameByChunkID[r.Chunk.ID] = doc.OriginalName
		}
	}

	passed, _ := filterResults(allResults, 0.3)
	if len(passed) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<project_context>")
	for _, r := range passed {
		docName := docNameByChunkID[r.Chunk.ID]
		sb.WriteString(fmt.Sprintf(
			"<document source=%q relevance=\"%.2f\">%s</document>",
			docName,
			r.Score,
			r.Chunk.Text,
		))
	}
	sb.WriteString("</project_context>")

	return sb.String()
}

// performCodeReview runs the full AI code review pipeline, emitting SSE events.
func performCodeReview(w http.ResponseWriter, apiKey string, docStore *DocumentStore, prNumber int) {
	emitStep := func(step, status string) {
		sseWrite(w, map[string]any{
			"type":   "review_step",
			"step":   step,
			"status": status,
		})
	}

	// Step 1: fetch diff.
	emitStep("diff", "running")
	diff, err := getPRDiff(prNumber)
	if err != nil {
		emitStep("diff", "error")
		sseWrite(w, map[string]any{"type": "error", "message": err.Error()})
		sseWrite(w, map[string]any{"type": "done"})
		return
	}
	emitStep("diff", "done")

	// Step 2: RAG context enrichment.
	emitStep("rag", "running")
	ragContext := buildReviewRAGContext(apiKey, docStore, diff)
	if ragContext == "" {
		emitStep("rag", "skipped")
	} else {
		emitStep("rag", "done")
	}

	// Step 3: stream Claude analysis.
	emitStep("analyze", "running")

	systemPrompt := `You are an expert code reviewer. Analyze the following git diff and provide a thorough code review.

Structure your review as markdown with these sections:
## Summary
Brief overview of changes.

## Potential Bugs
List any bugs, logic errors, or runtime issues found. If none, say "No bugs found."

## Architecture Issues
Any structural, design, or maintainability concerns. If none, say "No issues found."

## Recommendations
Specific improvement suggestions.

## Verdict
APPROVE or REQUEST_CHANGES with brief justification.

If project documentation context is provided, also check consistency with project conventions and architecture.
Be specific — reference file names and line numbers from the diff.`

	userContent := "<diff>\n" + diff + "\n</diff>"
	if ragContext != "" {
		userContent += "\n\n" + ragContext
	}

	reqBody, _ := json.Marshal(map[string]any{
		"model":      ModelSonnet,
		"max_tokens": 4096,
		"stream":     true,
		"temperature": 0,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": userContent},
		},
	})

	httpReq, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		emitStep("analyze", "error")
		sseWrite(w, map[string]any{"type": "error", "message": err.Error()})
		sseWrite(w, map[string]any{"type": "done"})
		return
	}
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		emitStep("analyze", "error")
		sseWrite(w, map[string]any{"type": "error", "message": err.Error()})
		sseWrite(w, map[string]any{"type": "done"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		emitStep("analyze", "error")
		sseWrite(w, map[string]any{"type": "error", "message": fmt.Sprintf("Claude API error: %d", resp.StatusCode)})
		sseWrite(w, map[string]any{"type": "done"})
		return
	}

	reviewText, _, streamErr := readStream(resp.Body, func(token string) {
		sseWrite(w, map[string]any{"type": "text_delta", "text": token})
	})
	if streamErr != nil {
		emitStep("analyze", "error")
		sseWrite(w, map[string]any{"type": "error", "message": streamErr.Error()})
		sseWrite(w, map[string]any{"type": "done"})
		return
	}

	emitStep("analyze", "done")

	// Step 4: post comment to GitHub.
	emitStep("comment", "running")

	commentBody := "## AI Code Review\n\n" + reviewText + "\n\n---\n*Generated by AI Code Review*"
	if commentErr := postPRComment(prNumber, commentBody); commentErr != nil {
		emitStep("comment", "error")
		sseWrite(w, map[string]any{"type": "error", "message": commentErr.Error()})
		sseWrite(w, map[string]any{"type": "done"})
		return
	}

	emitStep("comment", "done")
	sseWrite(w, map[string]any{"type": "done"})
}

// handleListPRs handles GET /api/review/prs — returns open pull requests as JSON.
func handleListPRs(w http.ResponseWriter, r *http.Request) {
	prs, err := listOpenPRs()
	if err != nil {
		// Return empty array on error to keep the frontend working.
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]pullRequest{})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prs)
}

// handleRunReview handles POST /api/review/run — streams the code review via SSE.
func handleRunReview(w http.ResponseWriter, r *http.Request, apiKey string, docStore *DocumentStore) {
	var req reviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.PRNumber <= 0 {
		http.Error(w, "pr_number is required", http.StatusBadRequest)
		return
	}

	sseSetup(w)
	performCodeReview(w, apiKey, docStore, req.PRNumber)
}
