package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

const hnFirebaseBase = "https://hacker-news.firebaseio.com/v0"

// RunTask dispatches to the appropriate runner based on task type.
func RunTask(ctx context.Context, store *Store, taskID string) {
	task, ok := store.Get(taskID)
	if !ok {
		return
	}

	switch task.Type {
	case TypeReminder:
		runReminder(ctx, store, task)
	case TypeURLMonitor:
		runURLMonitor(ctx, store, task)
	case TypeHNDigest:
		runHNDigest(ctx, store, task)
	case TypePipeline:
		runPipeline(ctx, store, task)
	}
}

// runReminder fires once after the given delay.
func runReminder(ctx context.Context, store *Store, task *Task) {
	delayStr := task.Params["delay"]
	if delayStr == "" {
		delayStr = task.Interval
	}

	delay, err := time.ParseDuration(delayStr)
	if err != nil {
		store.AppendResult(task.ID, TaskResult{
			Timestamp: time.Now(),
			Success:   false,
			Error:     fmt.Sprintf("invalid delay %q: %v", delayStr, err),
		})
		return
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		msg := task.Params["message"]
		store.AppendResult(task.ID, TaskResult{
			Timestamp: time.Now(),
			Success:   true,
			Data:      fmt.Sprintf("⏰ Reminder: %s", msg),
		})
		store.UpdateStatus(task.ID, StatusDone)
		store.CancelRunner(task.ID)
	}
}

// runURLMonitor polls a URL at the task's interval.
func runURLMonitor(ctx context.Context, store *Store, task *Task) {
	interval, err := time.ParseDuration(task.Interval)
	if err != nil {
		store.AppendResult(task.ID, TaskResult{
			Timestamp: time.Now(),
			Success:   false,
			Error:     fmt.Sprintf("invalid interval %q: %v", task.Interval, err),
		})
		return
	}

	url := task.Params["url"]
	if url == "" {
		store.AppendResult(task.ID, TaskResult{
			Timestamp: time.Now(),
			Success:   false,
			Error:     "missing url param",
		})
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			statusCode, responseTime, bodySize, fetchErr := fetchURL(ctx, url)
			result := TaskResult{Timestamp: time.Now()}
			if fetchErr != nil {
				result.Success = false
				result.Error = fetchErr.Error()
				result.Data = fmt.Sprintf("✗ Error: %s", fetchErr.Error())
			} else {
				result.Success = statusCode >= 200 && statusCode < 400
				sizeKB := float64(bodySize) / 1024.0
				result.Data = fmt.Sprintf("✓ %d OK | %dms | %.1f KB",
					statusCode, responseTime.Milliseconds(), sizeKB)
			}
			store.AppendResult(task.ID, result)
		}
	}
}

// runHNDigest fetches HN top stories at the task's interval.
func runHNDigest(ctx context.Context, store *Store, task *Task) {
	interval, err := time.ParseDuration(task.Interval)
	if err != nil {
		store.AppendResult(task.ID, TaskResult{
			Timestamp: time.Now(),
			Success:   false,
			Error:     fmt.Sprintf("invalid interval %q: %v", task.Interval, err),
		})
		return
	}

	count := 5
	if v := task.Params["count"]; v != "" {
		if n, parseErr := fmt.Sscanf(v, "%d", &count); n != 1 || parseErr != nil {
			count = 5
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			digest, fetchErr := fetchHNTopStories(ctx, count)
			result := TaskResult{Timestamp: time.Now()}
			if fetchErr != nil {
				result.Success = false
				result.Error = fetchErr.Error()
			} else {
				result.Success = true
				result.Data = digest
			}
			store.AppendResult(task.ID, result)
		}
	}
}

// pipelineClient has a longer timeout for pipeline runs that involve AI summarization.
var pipelineClient = &http.Client{Timeout: 120 * time.Second}

// runPipeline fires once after the given delay, calling pipe_run via backend API.
func runPipeline(ctx context.Context, store *Store, task *Task) {
	delayStr := task.Params["delay"]
	if delayStr == "" {
		delayStr = task.Interval
	}

	delay, err := time.ParseDuration(delayStr)
	if err != nil {
		store.AppendResult(task.ID, TaskResult{
			Timestamp: time.Now(),
			Success:   false,
			Error:     fmt.Sprintf("invalid delay %q: %v", delayStr, err),
		})
		return
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		backendURL := task.Params["backend_url"]
		if backendURL == "" {
			backendURL = "http://localhost:8080"
		}

		query := task.Params["query"]
		count := task.Params["count"]
		if count == "" {
			count = "5"
		}

		body := fmt.Sprintf(`{"server":"pipeline","tool":"pipe_run","arguments":{"query":%q,"count":%s}}`,
			query, count)

		req, reqErr := http.NewRequestWithContext(ctx, "POST",
			backendURL+"/api/mcp/tools/call",
			strings.NewReader(body))
		if reqErr != nil {
			store.AppendResult(task.ID, TaskResult{
				Timestamp: time.Now(),
				Success:   false,
				Error:     fmt.Sprintf("failed to create request: %v", reqErr),
			})
			store.UpdateStatus(task.ID, StatusDone)
			store.CancelRunner(task.ID)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, respErr := pipelineClient.Do(req)
		if respErr != nil {
			store.AppendResult(task.ID, TaskResult{
				Timestamp: time.Now(),
				Success:   false,
				Error:     fmt.Sprintf("HTTP request failed: %v", respErr),
			})
			store.UpdateStatus(task.ID, StatusDone)
			store.CancelRunner(task.ID)
			return
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		success := resp.StatusCode >= 200 && resp.StatusCode < 400

		store.AppendResult(task.ID, TaskResult{
			Timestamp: time.Now(),
			Success:   success,
			Data:      string(respBody),
		})
		store.UpdateStatus(task.ID, StatusDone)
		store.CancelRunner(task.ID)
	}
}

// fetchURL performs an HTTP GET and returns status code, response time, body size.
func fetchURL(ctx context.Context, url string) (statusCode int, responseTime time.Duration, bodySize int, err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, 0, 0, err
	}

	start := time.Now()
	resp, err := httpClient.Do(req)
	responseTime = time.Since(start)
	if err != nil {
		return 0, responseTime, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, responseTime, 0, err
	}

	return resp.StatusCode, responseTime, len(body), nil
}

// fetchHNTopStories fetches the top N stories from HN Firebase API and formats a digest.
func fetchHNTopStories(ctx context.Context, count int) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", hnFirebaseBase+"/topstories.json", nil)
	if err != nil {
		return "", err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var ids []int
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		return "", err
	}

	if len(ids) > count {
		ids = ids[:count]
	}

	type hnItem struct {
		Title string `json:"title"`
		URL   string `json:"url"`
		Score int    `json:"score"`
	}

	result := fmt.Sprintf("Top %d HN Stories:\n", count)
	for i, id := range ids {
		itemReq, err := http.NewRequestWithContext(ctx, "GET",
			fmt.Sprintf("%s/item/%d.json", hnFirebaseBase, id), nil)
		if err != nil {
			continue
		}

		itemResp, err := httpClient.Do(itemReq)
		if err != nil {
			continue
		}

		var item hnItem
		_ = json.NewDecoder(itemResp.Body).Decode(&item)
		itemResp.Body.Close()

		url := item.URL
		if url == "" {
			url = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", id)
		}
		result += fmt.Sprintf("%d. [%d] %s (%s)\n", i+1, item.Score, item.Title, url)
	}

	return result, nil
}
