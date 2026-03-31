# MCP Servers

All MCP servers use stdio transport, `github.com/mark3labs/mcp-go` library, and are built as standalone Go binaries.

Config file: `.mcp_servers.json` (Claude Desktop format):
```json
{
  "mcpServers": {
    "server_name": {
      "command": "./mcp-servers/name/name",
      "args": [],
      "env": {}
    }
  }
}
```

Tool naming in chat: `server__toolname` (double underscore separator).

## hackernews

HackerNews API integration. No API keys required.

**Tools:**

| Tool | Params | Description |
|------|--------|-------------|
| `hn_top_stories` | `count` (1-30, default 10) | Top stories with titles, URLs, scores |
| `hn_get_item` | `id` (required) | Get story/comment/job by ID |
| `hn_get_user` | `username` (required) | Get user profile |
| `hn_search` | `query` (required), `tags`, `count` | Search via Algolia |

Build: `go build -o mcp-servers/hackernews/hackernews ./mcp-servers/hackernews/`

## scheduler

Task scheduler with 4 job types. JSON persistence, goroutine-based execution.

**Tools:**

| Tool | Params | Description |
|------|--------|-------------|
| `sched_create` | `type`, `name`, `interval`, `config` | Create scheduled task |
| `sched_list` | - | List all tasks |
| `sched_status` | `id` | Get task status |
| `sched_delete` | `id` | Delete task |
| `sched_pause` | `id` | Pause task |
| `sched_resume` | `id` | Resume task |

**Job types:** `reminder` (one-shot), `url_monitor` (periodic HTTP GET), `hn_digest` (periodic HN fetch), `pipeline` (one-shot pipeline trigger).

Build: `go build -o mcp-servers/scheduler/scheduler ./mcp-servers/scheduler/`

## pipeline

HN search -> Claude Haiku summary -> file save. Async execution.

**Tools:**

| Tool | Params | Description |
|------|--------|-------------|
| `pipe_search` | `query`, `count` | Search HN via Algolia |
| `pipe_summarize` | `text`, `style` | Summarize with Claude Haiku |
| `pipe_save` | `content`, `filename` | Save to file |
| `pipe_run` | `query`, `count`, `style`, `filename` | Run full pipeline async |
| `pipe_status` | `id` | Check pipeline run status |
| `pipe_list` | - | List all pipeline runs |
| `pipe_delete` | `id` | Delete pipeline run |

Requires: `ANTHROPIC_API_KEY` env var for summarization.

Build: `go build -o mcp-servers/pipeline/pipeline ./mcp-servers/pipeline/`

## devtools

Project context tools for the developer assistant. Read-only operations.

**Tools:**

| Tool | Params | Description |
|------|--------|-------------|
| `dev_git_branch` | - | Get current git branch and list all branches |
| `dev_git_status` | - | Get git status (modified/untracked files) |
| `dev_git_log` | `count` (default 10) | Recent commit log |
| `dev_list_files` | `path` (default "."), `pattern` | List files in directory |
| `dev_read_file` | `path` (required), `max_lines` (default 100) | Read file contents |

Build: `go build -o mcp-servers/devtools/devtools ./mcp-servers/devtools/`

## Creating a New MCP Server

Minimal template:

```go
package main

import (
    "context"
    "fmt"
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

func main() {
    s := server.NewMCPServer("my-server", "1.0.0",
        server.WithToolCapabilities(true),
    )

    s.AddTool(
        mcp.NewTool("my_tool",
            mcp.WithDescription("Description"),
            mcp.WithString("param", mcp.Required(), mcp.Description("...")),
        ),
        handleMyTool,
    )

    if err := server.ServeStdio(s); err != nil {
        fmt.Printf("Server error: %v\n", err)
    }
}

func handleMyTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    val, _ := req.RequireString("param")
    return mcp.NewToolResultText("Result: " + val), nil
}
```

Add to `.mcp_servers.json`, build with `go build -o mcp-servers/name/name ./mcp-servers/name/`.
