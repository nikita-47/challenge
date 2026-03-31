package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func projectDir() string {
	if d := os.Getenv("PROJECT_DIR"); d != "" {
		return d
	}
	return "."
}

func main() {
	s := server.NewMCPServer("devtools", "1.0.0",
		server.WithToolCapabilities(true),
	)

	s.AddTool(
		mcp.NewTool("dev_git_branch",
			mcp.WithDescription("Get the current git branch and list all branches."),
		),
		handleGitBranch,
	)

	s.AddTool(
		mcp.NewTool("dev_git_status",
			mcp.WithDescription("Get git status showing modified, staged, and untracked files."),
		),
		handleGitStatus,
	)

	s.AddTool(
		mcp.NewTool("dev_git_log",
			mcp.WithDescription("Get recent git commit log."),
			mcp.WithNumber("count", mcp.Description("Number of commits to show (default 10, max 50)")),
		),
		handleGitLog,
	)

	s.AddTool(
		mcp.NewTool("dev_list_files",
			mcp.WithDescription("List files in a directory. Returns file paths relative to project root."),
			mcp.WithString("path", mcp.Description("Directory path relative to project root (default: '.')")),
			mcp.WithString("pattern", mcp.Description("Glob pattern to filter files (e.g. '*.go', '*.vue')")),
		),
		handleListFiles,
	)

	s.AddTool(
		mcp.NewTool("dev_read_file",
			mcp.WithDescription("Read the contents of a file. Returns text content with line numbers."),
			mcp.WithString("path", mcp.Required(), mcp.Description("File path relative to project root")),
			mcp.WithNumber("max_lines", mcp.Description("Maximum lines to read (default 100, max 500)")),
		),
		handleReadFile,
	)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func runGit(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = projectDir()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %s", strings.Join(args, " "), string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

func handleGitBranch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	current, err := runGit(ctx, "branch", "--show-current")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get current branch: %v", err)), nil
	}

	branches, err := runGit(ctx, "branch", "-a", "--format=%(refname:short)")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list branches: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Current branch: %s\n\nAll branches:\n", current))
	for _, b := range strings.Split(branches, "\n") {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		marker := "  "
		if b == current {
			marker = "* "
		}
		sb.WriteString(marker + b + "\n")
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func handleGitStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	status, err := runGit(ctx, "status", "--short")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get status: %v", err)), nil
	}

	if status == "" {
		return mcp.NewToolResultText("Working tree clean — no changes."), nil
	}

	return mcp.NewToolResultText(status), nil
}

func handleGitLog(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	count := 10
	if v, err := req.RequireInt("count"); err == nil && v > 0 {
		count = v
	}
	if count > 50 {
		count = 50
	}

	log, err := runGit(ctx, "log", fmt.Sprintf("--oneline"), fmt.Sprintf("-%d", count))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get log: %v", err)), nil
	}

	return mcp.NewToolResultText(log), nil
}

func handleListFiles(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dirPath := "."
	if v, err := req.RequireString("path"); err == nil && v != "" {
		dirPath = v
	}

	// Prevent path traversal.
	if strings.Contains(dirPath, "..") {
		return mcp.NewToolResultError("Path traversal not allowed"), nil
	}

	fullPath := filepath.Join(projectDir(), dirPath)

	pattern := ""
	if v, err := req.RequireString("pattern"); err == nil && v != "" {
		pattern = v
	}

	var files []string
	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip hidden dirs and common ignores.
		name := info.Name()
		if info.IsDir() {
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}

		rel, _ := filepath.Rel(projectDir(), path)

		if pattern != "" {
			matched, _ := filepath.Match(pattern, name)
			if !matched {
				return nil
			}
		}

		files = append(files, rel)
		if len(files) >= 200 {
			return fmt.Errorf("limit reached")
		}
		return nil
	})

	if err != nil && len(files) == 0 {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list files: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Files in %s", dirPath))
	if pattern != "" {
		sb.WriteString(fmt.Sprintf(" (pattern: %s)", pattern))
	}
	sb.WriteString(fmt.Sprintf(" (%d files):\n\n", len(files)))
	for _, f := range files {
		sb.WriteString(f + "\n")
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func handleReadFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("path is required"), nil
	}

	// Prevent path traversal.
	if strings.Contains(path, "..") {
		return mcp.NewToolResultError("Path traversal not allowed"), nil
	}

	maxLines := 100
	if v, err := req.RequireInt("max_lines"); err == nil && v > 0 {
		maxLines = v
	}
	if maxLines > 500 {
		maxLines = 500
	}

	fullPath := filepath.Join(projectDir(), path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Cannot read file: %v", err)), nil
	}

	lines := strings.Split(string(data), "\n")
	truncated := false
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		truncated = true
	}

	var sb strings.Builder
	for i, line := range lines {
		sb.WriteString(fmt.Sprintf("%4d | %s\n", i+1, line))
	}
	if truncated {
		sb.WriteString(fmt.Sprintf("\n... truncated at %d lines (file has more)\n", maxLines))
	}

	return mcp.NewToolResultText(sb.String()), nil
}
