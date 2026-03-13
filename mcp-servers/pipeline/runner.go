package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

const (
	hnAlgoliaBase = "https://hn.algolia.com/api/v1"
	claudeAPIBase = "https://api.anthropic.com/v1"
)

// RunPipeline executes the full search → summarize → save pipeline.
// It is designed to be called in a goroutine; state is tracked via the Store.
func RunPipeline(store *Store, id string) {
	run, ok := store.Get(id)
	if !ok {
		return
	}

	store.UpdateStatus(id, StatusRunning)

	count := run.Count
	if count <= 0 {
		count = 5
	}

	// Step 1: search
	searchOutput, err := runSearchStep(store, id, run.Query, count)
	if err != nil {
		store.UpdateStatus(id, StatusError)
		return
	}

	// Step 2: summarize
	summary, err := runSummarizeStep(store, id, searchOutput)
	if err != nil {
		store.UpdateStatus(id, StatusError)
		return
	}

	// Step 3: save
	if err := runSaveStep(store, id, run.Query, searchOutput, summary); err != nil {
		store.UpdateStatus(id, StatusError)
		return
	}

	store.UpdateStatus(id, StatusDone)
}

// runSearchStep performs the HN Algolia search and records results.
func runSearchStep(store *Store, id string, query string, count int) (string, error) {
	store.UpdateStep(id, "search", func(s *PipelineStep) {
		s.Status = StatusRunning
		s.StartedAt = time.Now()
		s.Error = "" // clear temp count storage
	})

	output, err := SearchHN(context.Background(), query, count)
	if err != nil {
		store.UpdateStep(id, "search", func(s *PipelineStep) {
			s.Status = StatusError
			s.FinishedAt = time.Now()
			s.Error = err.Error()
		})
		return "", err
	}

	preview := output
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}

	store.UpdateStep(id, "search", func(s *PipelineStep) {
		s.Status = StatusDone
		s.FinishedAt = time.Now()
		s.Output = preview
	})

	return output, nil
}

// runSummarizeStep calls the Claude API to summarize search results.
func runSummarizeStep(store *Store, id string, searchOutput string) (string, error) {
	store.UpdateStep(id, "summarize", func(s *PipelineStep) {
		s.Status = StatusRunning
		s.StartedAt = time.Now()
	})

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		err := fmt.Errorf("ANTHROPIC_API_KEY is not set")
		store.UpdateStep(id, "summarize", func(s *PipelineStep) {
			s.Status = StatusError
			s.FinishedAt = time.Now()
			s.Error = err.Error()
		})
		return "", err
	}

	summary, err := SummarizeWithClaude(context.Background(), apiKey, searchOutput)
	if err != nil {
		store.UpdateStep(id, "summarize", func(s *PipelineStep) {
			s.Status = StatusError
			s.FinishedAt = time.Now()
			s.Error = err.Error()
		})
		return "", err
	}

	preview := summary
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}

	store.UpdateStep(id, "summarize", func(s *PipelineStep) {
		s.Status = StatusDone
		s.FinishedAt = time.Now()
		s.Output = preview
	})

	return summary, nil
}

// runSaveStep writes the pipeline output to a markdown file.
func runSaveStep(store *Store, id string, query string, searchOutput string, summary string) error {
	store.UpdateStep(id, "save", func(s *PipelineStep) {
		s.Status = StatusRunning
		s.StartedAt = time.Now()
	})

	outputPath, err := SavePipelineOutput(id, query, searchOutput, summary)
	if err != nil {
		store.UpdateStep(id, "save", func(s *PipelineStep) {
			s.Status = StatusError
			s.FinishedAt = time.Now()
			s.Error = err.Error()
		})
		return err
	}

	store.SetOutputFile(id, outputPath)
	store.UpdateStep(id, "save", func(s *PipelineStep) {
		s.Status = StatusDone
		s.FinishedAt = time.Now()
		s.Output = outputPath
	})

	return nil
}

// SearchHN queries the HN Algolia API and returns formatted results.
func SearchHN(ctx context.Context, query string, count int) (string, error) {
	if count <= 0 {
		count = 5
	}

	reqURL := fmt.Sprintf("%s/search?query=%s&tags=story&hitsPerPage=%d",
		hnAlgoliaBase, url.QueryEscape(query), count)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var result struct {
		Hits []struct {
			Title     string `json:"title"`
			URL       string `json:"url"`
			Points    int    `json:"points"`
			Author    string `json:"author"`
			ObjectID  string `json:"objectID"`
			NumComments int  `json:"num_comments"`
		} `json:"hits"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(result.Hits) == 0 {
		return fmt.Sprintf("No results found for query: %s", query), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("HackerNews search results for %q (%d hits):\n\n", query, len(result.Hits)))

	for i, hit := range result.Hits {
		link := hit.URL
		if link == "" {
			link = fmt.Sprintf("https://news.ycombinator.com/item?id=%s", hit.ObjectID)
		}
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, hit.Title))
		sb.WriteString(fmt.Sprintf("   Points: %d | Comments: %d | By: %s\n", hit.Points, hit.NumComments, hit.Author))
		sb.WriteString(fmt.Sprintf("   URL: %s\n\n", link))
	}

	return sb.String(), nil
}

// SummarizeWithClaude calls the Claude API to produce a concise summary.
func SummarizeWithClaude(ctx context.Context, apiKey string, text string) (string, error) {
	reqBody := map[string]interface{}{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 1024,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": "Summarize the following search results concisely:\n\n" + text,
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", claudeAPIBase+"/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	for _, block := range apiResp.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}

	return "", fmt.Errorf("no text content in API response")
}

// SavePipelineOutput writes a markdown file with search results and summary.
// Returns the path of the written file.
func SavePipelineOutput(id string, query string, searchOutput string, summary string) (string, error) {
	outputDir := outputDirPath()
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	filename := filepath.Join(outputDir, id+".md")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Pipeline Run: %s\n\n", id))
	sb.WriteString(fmt.Sprintf("**Query:** %s  \n", query))
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format(time.RFC3339)))
	sb.WriteString("---\n\n")
	sb.WriteString("## Summary\n\n")
	sb.WriteString(summary)
	sb.WriteString("\n\n---\n\n")
	sb.WriteString("## Raw Search Results\n\n")
	sb.WriteString(searchOutput)

	if err := os.WriteFile(filename, []byte(sb.String()), 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return filename, nil
}
