package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var store *Store

func main() {
	store = NewStore(dataFilePath())
	if err := store.Load(); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Warning: could not load pipeline data: %v\n", err)
	}

	s := server.NewMCPServer("pipeline", "1.0.0",
		server.WithToolCapabilities(true),
	)

	s.AddTool(
		mcp.NewTool("pipe_search",
			mcp.WithDescription("Search HackerNews for stories matching a query using the Algolia API."),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
			mcp.WithNumber("count", mcp.Description("Number of results to return (default 5)")),
		),
		handleSearch,
	)

	s.AddTool(
		mcp.NewTool("pipe_summarize",
			mcp.WithDescription("Summarize text using the Claude API (claude-3-5-haiku)."),
			mcp.WithString("text", mcp.Required(), mcp.Description("Text to summarize")),
		),
		handleSummarize,
	)

	s.AddTool(
		mcp.NewTool("pipe_save",
			mcp.WithDescription("Save content to a file in pipeline_output/."),
			mcp.WithString("content", mcp.Required(), mcp.Description("Content to save")),
			mcp.WithString("filename", mcp.Required(), mcp.Description("Output filename (without directory)")),
		),
		handleSave,
	)

	s.AddTool(
		mcp.NewTool("pipe_run",
			mcp.WithDescription("Run the full search → summarize → save pipeline asynchronously. Returns a run ID immediately."),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search query for HackerNews")),
			mcp.WithNumber("count", mcp.Description("Number of search results (default 5)")),
			mcp.WithString("filename", mcp.Description("Output filename prefix (optional, auto-generated from run ID if omitted)")),
		),
		handleRun,
	)

	s.AddTool(
		mcp.NewTool("pipe_status",
			mcp.WithDescription("Get the status of a pipeline run. If id is omitted, returns the most recent run."),
			mcp.WithString("id", mcp.Description("Pipeline run ID (optional)")),
		),
		handleStatus,
	)

	s.AddTool(
		mcp.NewTool("pipe_list",
			mcp.WithDescription("List all pipeline runs with their overall status."),
		),
		handleList,
	)

	s.AddTool(
		mcp.NewTool("pipe_delete",
			mcp.WithDescription("Delete a pipeline run by ID. Cannot delete a currently running pipeline."),
			mcp.WithString("id", mcp.Required(), mcp.Description("Pipeline run ID to delete")),
		),
		handleDelete,
	)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
	}

	_ = store.Save()
}

// dataFilePath determines the storage path for pipeline_data.json.
// Priority: PIPELINE_DATA_PATH env var > directory of the executable.
func dataFilePath() string {
	if path := os.Getenv("PIPELINE_DATA_PATH"); path != "" {
		return path
	}

	exe, err := exec.LookPath(os.Args[0])
	if err != nil {
		exe = os.Args[0]
	}

	abs, err := filepath.Abs(exe)
	if err != nil {
		return "pipeline_data.json"
	}

	return filepath.Join(filepath.Dir(abs), "pipeline_data.json")
}

// outputDirPath returns the directory for pipeline output files,
// placed alongside the executable.
func outputDirPath() string {
	exe, err := exec.LookPath(os.Args[0])
	if err != nil {
		exe = os.Args[0]
	}

	abs, err := filepath.Abs(exe)
	if err != nil {
		return "pipeline_output"
	}

	return filepath.Join(filepath.Dir(abs), "pipeline_output")
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func handleSearch(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("query is required"), nil
	}

	count := 5
	if v, countErr := req.RequireInt("count"); countErr == nil && v > 0 {
		count = v
	}

	output, searchErr := SearchHN(context.Background(), query, count)
	if searchErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", searchErr)), nil
	}

	return mcp.NewToolResultText(output), nil
}

func handleSummarize(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	text, err := req.RequireString("text")
	if err != nil {
		return mcp.NewToolResultError("text is required"), nil
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return mcp.NewToolResultError("ANTHROPIC_API_KEY environment variable is not set"), nil
	}

	summary, sumErr := SummarizeWithClaude(context.Background(), apiKey, text)
	if sumErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("summarize failed: %v", sumErr)), nil
	}

	return mcp.NewToolResultText(summary), nil
}

func handleSave(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content, err := req.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError("content is required"), nil
	}

	filename, err := req.RequireString("filename")
	if err != nil {
		return mcp.NewToolResultError("filename is required"), nil
	}

	outputDir := outputDirPath()
	if mkErr := os.MkdirAll(outputDir, 0755); mkErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create output dir: %v", mkErr)), nil
	}

	outPath := filepath.Join(outputDir, filepath.Base(filename))
	if writeErr := os.WriteFile(outPath, []byte(content), 0644); writeErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to write file: %v", writeErr)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Saved to: %s", outPath)), nil
}

func handleRun(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("query is required"), nil
	}

	count := 5
	if v, countErr := req.RequireInt("count"); countErr == nil && v > 0 {
		count = v
	}

	id, idErr := generateID()
	if idErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to generate ID: %v", idErr)), nil
	}

	run := &PipelineRun{
		ID:        id,
		Query:     query,
		Source:    "hackernews",
		Count:     count,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		Steps: []PipelineStep{
			{Name: "search", Status: StatusPending},
			{Name: "summarize", Status: StatusPending},
			{Name: "save", Status: StatusPending},
		},
	}

	store.Add(run)
	go RunPipeline(store, id)

	return mcp.NewToolResultText(fmt.Sprintf(
		"Pipeline started: %s\nQuery: %s\nCount: %d\n\nUse pipe_status id=%s to check progress.",
		id, query, count, id,
	)), nil
}

func handleStatus(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, _ := req.RequireString("id")

	var run *PipelineRun
	if id == "" {
		run = store.Latest()
		if run == nil {
			return mcp.NewToolResultText("No pipeline runs found."), nil
		}
	} else {
		var ok bool
		run, ok = store.Get(id)
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("run %q not found", id)), nil
		}
	}

	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal run: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

func handleDelete(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}

	run, ok := store.Get(id)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("run %q not found", id)), nil
	}

	if run.Status == StatusRunning {
		return mcp.NewToolResultError(fmt.Sprintf("run %q is currently running, cannot delete", id)), nil
	}

	if !store.Delete(id) {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete run %q", id)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Deleted run %s", id)), nil
}

func handleList(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	runs := store.List()
	if len(runs) == 0 {
		return mcp.NewToolResultText("[]"), nil
	}

	data, err := json.MarshalIndent(runs, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal runs: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}
