package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var (
	store      *Store
	rootCtx    context.Context
	rootCancel context.CancelFunc
)

func main() {
	store = NewStore(dataFilePath())
	if err := store.Load(); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Warning: could not load scheduler data: %v\n", err)
	}

	rootCtx, rootCancel = context.WithCancel(context.Background())

	// Restore active tasks from persisted state.
	for _, task := range store.List() {
		if task.Status == StatusActive {
			ctx, cancel := context.WithCancel(rootCtx)
			store.SetRunner(task.ID, cancel)
			go RunTask(ctx, store, task.ID)
		}
	}

	s := server.NewMCPServer("scheduler", "1.0.0",
		server.WithToolCapabilities(true),
	)

	s.AddTool(
		mcp.NewTool("sched_create",
			mcp.WithDescription("Create a new scheduled task. Types: reminder (one-shot after delay), url_monitor (periodic HTTP check), hn_digest (periodic HackerNews digest), pipeline (one-shot delayed pipeline run via backend)."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Human-readable task name")),
			mcp.WithString("type", mcp.Required(), mcp.Description("Task type: reminder, url_monitor, hn_digest, or pipeline")),
			mcp.WithString("interval", mcp.Description("Repeat interval for url_monitor/hn_digest (e.g. 5m, 1h, 30s)")),
			mcp.WithString("url", mcp.Description("URL to monitor (required for url_monitor)")),
			mcp.WithString("message", mcp.Description("Reminder message (required for reminder)")),
			mcp.WithString("delay", mcp.Description("One-shot delay for reminder (e.g. 10s, 5m, 1h)")),
			mcp.WithNumber("count", mcp.Description("Number of HN stories in digest (default 5, for hn_digest)")),
		mcp.WithString("query", mcp.Description("Search query for pipeline task (required for pipeline type)")),
		mcp.WithString("backend_url", mcp.Description("Backend API URL for pipeline execution (default http://localhost:8080)")),
		),
		handleCreate,
	)

	s.AddTool(
		mcp.NewTool("sched_list",
			mcp.WithDescription("List all scheduled tasks with their status and last run time."),
		),
		handleList,
	)

	s.AddTool(
		mcp.NewTool("sched_status",
			mcp.WithDescription("Get detailed status and recent results for a task."),
			mcp.WithString("id", mcp.Required(), mcp.Description("Task ID")),
			mcp.WithNumber("count", mcp.Description("Number of recent results to show (default 5)")),
		),
		handleStatus,
	)

	s.AddTool(
		mcp.NewTool("sched_delete",
			mcp.WithDescription("Delete a scheduled task and stop its execution."),
			mcp.WithString("id", mcp.Required(), mcp.Description("Task ID")),
		),
		handleDelete,
	)

	s.AddTool(
		mcp.NewTool("sched_pause",
			mcp.WithDescription("Pause an active scheduled task."),
			mcp.WithString("id", mcp.Required(), mcp.Description("Task ID")),
		),
		handlePause,
	)

	s.AddTool(
		mcp.NewTool("sched_resume",
			mcp.WithDescription("Resume a paused scheduled task."),
			mcp.WithString("id", mcp.Required(), mcp.Description("Task ID")),
		),
		handleResume,
	)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}

	rootCancel()
	_ = store.Save()
}

// dataFilePath determines the storage path for scheduler_data.json.
// Priority: SCHED_DATA_PATH env var > directory of the executable.
func dataFilePath() string {
	if path := os.Getenv("SCHED_DATA_PATH"); path != "" {
		return path
	}

	exe, err := exec.LookPath(os.Args[0])
	if err != nil {
		exe = os.Args[0]
	}

	abs, err := filepath.Abs(exe)
	if err != nil {
		return "scheduler_data.json"
	}

	return filepath.Join(filepath.Dir(abs), "scheduler_data.json")
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func handleCreate(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("name is required"), nil
	}

	taskType, err := req.RequireString("type")
	if err != nil {
		return mcp.NewToolResultError("type is required (reminder, url_monitor, hn_digest)"), nil
	}

	switch TaskType(taskType) {
	case TypeReminder, TypeURLMonitor, TypeHNDigest, TypePipeline:
		// valid
	default:
		return mcp.NewToolResultError(fmt.Sprintf("unknown task type %q; valid: reminder, url_monitor, hn_digest, pipeline", taskType)), nil
	}

	params := make(map[string]string)
	interval := ""

	switch TaskType(taskType) {
	case TypeReminder:
		msg, _ := req.RequireString("message")
		params["message"] = msg

		delay, _ := req.RequireString("delay")
		if delay == "" {
			// Fall back to interval field as delay.
			delay, _ = req.RequireString("interval")
		}
		if delay == "" {
			return mcp.NewToolResultError("delay is required for reminder (e.g. 10s, 5m)"), nil
		}
		if _, parseErr := time.ParseDuration(delay); parseErr != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid delay %q: %v", delay, parseErr)), nil
		}
		params["delay"] = delay
		interval = delay

	case TypeURLMonitor:
		url, _ := req.RequireString("url")
		if url == "" {
			return mcp.NewToolResultError("url is required for url_monitor"), nil
		}
		params["url"] = url

		interval, _ = req.RequireString("interval")
		if interval == "" {
			return mcp.NewToolResultError("interval is required for url_monitor (e.g. 5m)"), nil
		}
		if _, parseErr := time.ParseDuration(interval); parseErr != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid interval %q: %v", interval, parseErr)), nil
		}

	case TypeHNDigest:
		interval, _ = req.RequireString("interval")
		if interval == "" {
			return mcp.NewToolResultError("interval is required for hn_digest (e.g. 1h)"), nil
		}
		if _, parseErr := time.ParseDuration(interval); parseErr != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid interval %q: %v", interval, parseErr)), nil
		}

		count := 5
		if v, countErr := req.RequireInt("count"); countErr == nil && v > 0 {
			count = v
		}
		params["count"] = fmt.Sprintf("%d", count)

	case TypePipeline:
		query, _ := req.RequireString("query")
		if query == "" {
			return mcp.NewToolResultError("query is required for pipeline"), nil
		}
		params["query"] = query

		count := 5
		if v, countErr := req.RequireInt("count"); countErr == nil && v > 0 {
			count = v
		}
		params["count"] = fmt.Sprintf("%d", count)

		backendURL, _ := req.RequireString("backend_url")
		if backendURL == "" {
			backendURL = "http://localhost:8080"
		}
		params["backend_url"] = backendURL

		delay, _ := req.RequireString("delay")
		if delay == "" {
			delay, _ = req.RequireString("interval")
		}
		if delay == "" {
			return mcp.NewToolResultError("delay is required for pipeline (e.g. 10s, 5m)"), nil
		}
		if _, parseErr := time.ParseDuration(delay); parseErr != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid delay %q: %v", delay, parseErr)), nil
		}
		params["delay"] = delay
		interval = delay
	}

	id, idErr := generateID()
	if idErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to generate ID: %v", idErr)), nil
	}

	now := time.Now()
	task := &Task{
		ID:        id,
		Name:      name,
		Type:      TaskType(taskType),
		Status:    StatusActive,
		Interval:  interval,
		Params:    params,
		CreatedAt: now,
		NextRun:   now,
		Results:   []TaskResult{},
	}

	store.Add(task)

	taskCtx, cancel := context.WithCancel(rootCtx)
	store.SetRunner(id, cancel)
	go RunTask(taskCtx, store, id)

	return mcp.NewToolResultText(fmt.Sprintf(
		"✓ Task created: %s\n  Name: %s\n  Type: %s\n  Interval: %s",
		id, name, taskType, interval,
	)), nil
}

func handleList(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tasks := store.List()
	if len(tasks) == 0 {
		return mcp.NewToolResultText("No scheduled tasks."), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-8s  %-20s  %-12s  %-8s  %-10s  %s\n",
		"ID", "Name", "Type", "Status", "Interval", "Last Run"))
	sb.WriteString(strings.Repeat("-", 80) + "\n")

	for _, t := range tasks {
		lastRun := "never"
		if len(t.Results) > 0 {
			lastRun = t.Results[len(t.Results)-1].Timestamp.Format("15:04:05")
		}

		name := t.Name
		if len(name) > 20 {
			name = name[:17] + "..."
		}

		sb.WriteString(fmt.Sprintf("%-8s  %-20s  %-12s  %-8s  %-10s  %s\n",
			t.ID, name, string(t.Type), string(t.Status), t.Interval, lastRun))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func handleStatus(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}

	task, ok := store.Get(id)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("task %q not found", id)), nil
	}

	count := 5
	if v, countErr := req.RequireInt("count"); countErr == nil && v > 0 {
		count = v
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Task: %s (%s)\n", task.Name, task.ID))
	sb.WriteString(fmt.Sprintf("Type:     %s\n", task.Type))
	sb.WriteString(fmt.Sprintf("Status:   %s\n", task.Status))
	sb.WriteString(fmt.Sprintf("Interval: %s\n", task.Interval))
	sb.WriteString(fmt.Sprintf("Created:  %s\n", task.CreatedAt.Format(time.RFC3339)))

	if len(task.Params) > 0 {
		sb.WriteString("Params:\n")
		for k, v := range task.Params {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	// Aggregates for url_monitor.
	if task.Type == TypeURLMonitor && len(task.Results) > 0 {
		successCount := 0
		var totalMS int64
		var countWithTime int
		for _, r := range task.Results {
			if r.Success {
				successCount++
				// Parse response time from data string if present.
				var ms int64
				if n, _ := fmt.Sscanf(r.Data, "✓ %*d OK | %dms", &ms); n == 1 {
					totalMS += ms
					countWithTime++
				}
			}
		}
		uptimePct := float64(successCount) / float64(len(task.Results)) * 100.0
		sb.WriteString(fmt.Sprintf("\nAggregates (%d checks):\n", len(task.Results)))
		sb.WriteString(fmt.Sprintf("  Uptime: %.1f%%\n", uptimePct))
		if countWithTime > 0 {
			sb.WriteString(fmt.Sprintf("  Avg response: %dms\n", totalMS/int64(countWithTime)))
		}
	}

	// Recent results.
	results := task.Results
	if len(results) > count {
		results = results[len(results)-count:]
	}

	if len(results) > 0 {
		sb.WriteString(fmt.Sprintf("\nLast %d results:\n", len(results)))
		for _, r := range results {
			status := "✓"
			if !r.Success {
				status = "✗"
			}
			ts := r.Timestamp.Format("15:04:05")
			text := r.Data
			if r.Error != "" && text == "" {
				text = r.Error
			}
			sb.WriteString(fmt.Sprintf("  %s [%s] %s\n", status, ts, text))
		}
	} else {
		sb.WriteString("\nNo results yet.\n")
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func handleDelete(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}

	task, ok := store.Get(id)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("task %q not found", id)), nil
	}
	name := task.Name

	if !store.Delete(id) {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete task %q", id)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("✓ Task %s (%s) deleted", id, name)), nil
}

func handlePause(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}

	task, ok := store.Get(id)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("task %q not found", id)), nil
	}

	if task.Status != StatusActive {
		return mcp.NewToolResultError(fmt.Sprintf("task %q is not active (status: %s)", id, task.Status)), nil
	}

	name := task.Name
	store.CancelRunner(id)
	store.UpdateStatus(id, StatusPaused)

	return mcp.NewToolResultText(fmt.Sprintf("✓ Task %s (%s) paused", id, name)), nil
}

func handleResume(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}

	task, ok := store.Get(id)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("task %q not found", id)), nil
	}

	if task.Status != StatusPaused {
		return mcp.NewToolResultError(fmt.Sprintf("task %q is not paused (status: %s)", id, task.Status)), nil
	}

	name := task.Name
	store.UpdateStatus(id, StatusActive)

	taskCtx, cancel := context.WithCancel(rootCtx)
	store.SetRunner(id, cancel)
	go RunTask(taskCtx, store, id)

	return mcp.NewToolResultText(fmt.Sprintf("✓ Task %s (%s) resumed", id, name)), nil
}
