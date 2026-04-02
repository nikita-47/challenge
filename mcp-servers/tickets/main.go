package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var store *Store

func main() {
	store = NewStore(dataFilePath())
	if err := store.Load(); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Warning: could not load tickets data: %v\n", err)
	}

	s := server.NewMCPServer("tickets", "1.0.0",
		server.WithToolCapabilities(true),
	)

	s.AddTool(
		mcp.NewTool("ticket_create",
			mcp.WithDescription("Create a new support ticket. Returns the ticket ID and details."),
			mcp.WithString("title", mcp.Required(), mcp.Description("Short title describing the issue")),
			mcp.WithString("description", mcp.Required(), mcp.Description("Detailed description of the issue or request")),
			mcp.WithString("priority", mcp.Description("Priority level: low, medium, high (default: medium)")),
			mcp.WithString("user_email", mcp.Description("Optional user email for follow-up")),
		),
		handleCreate,
	)

	s.AddTool(
		mcp.NewTool("ticket_list",
			mcp.WithDescription("List support tickets. Optionally filter by status."),
			mcp.WithString("status", mcp.Description("Filter by status: open, closed, all (default: open)")),
		),
		handleList,
	)

	s.AddTool(
		mcp.NewTool("ticket_get",
			mcp.WithDescription("Get a support ticket by ID, including its full message history."),
			mcp.WithString("id", mcp.Required(), mcp.Description("Ticket ID")),
		),
		handleGet,
	)

	s.AddTool(
		mcp.NewTool("ticket_close",
			mcp.WithDescription("Close a support ticket with an optional resolution note."),
			mcp.WithString("id", mcp.Required(), mcp.Description("Ticket ID")),
			mcp.WithString("resolution", mcp.Description("Resolution summary or reason for closing")),
		),
		handleClose,
	)

	s.AddTool(
		mcp.NewTool("ticket_add_message",
			mcp.WithDescription("Add a message to an existing ticket's conversation history."),
			mcp.WithString("id", mcp.Required(), mcp.Description("Ticket ID")),
			mcp.WithString("role", mcp.Required(), mcp.Description("Message role: user or assistant")),
			mcp.WithString("content", mcp.Required(), mcp.Description("Message content")),
		),
		handleAddMessage,
	)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}

	_ = store.Save()
}

func dataFilePath() string {
	if path := os.Getenv("TICKETS_DATA_PATH"); path != "" {
		return path
	}

	exe, err := exec.LookPath(os.Args[0])
	if err != nil {
		exe = os.Args[0]
	}

	abs, err := filepath.Abs(exe)
	if err != nil {
		return "tickets_data.json"
	}

	return filepath.Join(filepath.Dir(abs), "tickets_data.json")
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func handleCreate(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title, err := req.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError("title is required"), nil
	}

	description, err := req.RequireString("description")
	if err != nil {
		return mcp.NewToolResultError("description is required"), nil
	}

	priority := "medium"
	if p, pErr := req.RequireString("priority"); pErr == nil && p != "" {
		switch p {
		case "low", "medium", "high":
			priority = p
		default:
			return mcp.NewToolResultError(fmt.Sprintf("invalid priority %q; valid: low, medium, high", p)), nil
		}
	}

	userEmail, _ := req.RequireString("user_email")

	id, idErr := generateID()
	if idErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to generate ID: %v", idErr)), nil
	}

	now := time.Now()
	ticket := &Ticket{
		ID:          id,
		Title:       title,
		Description: description,
		Status:      StatusOpen,
		Priority:    TicketPriority(priority),
		UserEmail:   userEmail,
		CreatedAt:   now,
		Messages:    []TicketMessage{},
	}

	store.Add(ticket)

	result, _ := json.MarshalIndent(ticket, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("Ticket created successfully:\n%s", string(result))), nil
}

func handleList(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	statusFilter := "open"
	if s, sErr := req.RequireString("status"); sErr == nil && s != "" {
		statusFilter = s
	}

	tickets := store.List()
	if len(tickets) == 0 {
		return mcp.NewToolResultText("No tickets found."), nil
	}

	var filtered []*Ticket
	for _, t := range tickets {
		if statusFilter == "all" || string(t.Status) == statusFilter {
			filtered = append(filtered, t)
		}
	}

	if len(filtered) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No tickets with status %q.", statusFilter)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-8s  %-30s  %-8s  %-8s  %-6s  %s\n",
		"ID", "Title", "Status", "Priority", "Msgs", "Created"))
	sb.WriteString(strings.Repeat("-", 90) + "\n")

	for _, t := range filtered {
		title := t.Title
		if len(title) > 30 {
			title = title[:27] + "..."
		}
		sb.WriteString(fmt.Sprintf("%-8s  %-30s  %-8s  %-8s  %-6d  %s\n",
			t.ID, title, string(t.Status), string(t.Priority),
			len(t.Messages), t.CreatedAt.Format("2006-01-02 15:04")))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func handleGet(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}

	ticket, ok := store.Get(id)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("ticket %q not found", id)), nil
	}

	result, _ := json.MarshalIndent(ticket, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func handleClose(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}

	ticket, ok := store.Get(id)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("ticket %q not found", id)), nil
	}

	if ticket.Status == StatusClosed {
		return mcp.NewToolResultError(fmt.Sprintf("ticket %q is already closed", id)), nil
	}

	resolution, _ := req.RequireString("resolution")

	now := time.Now()
	store.Close(id, resolution, now)

	return mcp.NewToolResultText(fmt.Sprintf("Ticket %s (%s) closed. Resolution: %s", id, ticket.Title, resolution)), nil
}

func handleAddMessage(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError("id is required"), nil
	}

	role, err := req.RequireString("role")
	if err != nil {
		return mcp.NewToolResultError("role is required"), nil
	}
	if role != "user" && role != "assistant" {
		return mcp.NewToolResultError("role must be 'user' or 'assistant'"), nil
	}

	content, err := req.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError("content is required"), nil
	}

	if _, ok := store.Get(id); !ok {
		return mcp.NewToolResultError(fmt.Sprintf("ticket %q not found", id)), nil
	}

	msg := TicketMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}

	store.AddMessage(id, msg)

	return mcp.NewToolResultText(fmt.Sprintf("Message added to ticket %s.", id)), nil
}
