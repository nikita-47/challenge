package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	firebaseBase = "https://hacker-news.firebaseio.com/v0"
	algoliaBase  = "https://hn.algolia.com/api/v1"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

func main() {
	s := server.NewMCPServer("hackernews", "1.0.0",
		server.WithToolCapabilities(true),
	)

	s.AddTool(
		mcp.NewTool("hn_top_stories",
			mcp.WithDescription("Get top stories from HackerNews. Returns titles, URLs, scores, and authors."),
			mcp.WithNumber("count", mcp.Description("Number of stories to return (1-30, default 10)")),
		),
		handleTopStories,
	)

	s.AddTool(
		mcp.NewTool("hn_get_item",
			mcp.WithDescription("Get a specific HackerNews item (story, comment, job, poll) by ID."),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("The item ID")),
		),
		handleGetItem,
	)

	s.AddTool(
		mcp.NewTool("hn_get_user",
			mcp.WithDescription("Get a HackerNews user profile by username."),
			mcp.WithString("username", mcp.Required(), mcp.Description("The username to look up")),
		),
		handleGetUser,
	)

	s.AddTool(
		mcp.NewTool("hn_search",
			mcp.WithDescription("Search HackerNews stories via Algolia. Returns matching stories with titles, URLs, scores, and points."),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
			mcp.WithString("tags", mcp.Description("Filter by tag: story, comment, ask_hn, show_hn, job")),
			mcp.WithNumber("count", mcp.Description("Number of results to return (1-30, default 5)")),
		),
		handleSearch,
	)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

// ── HackerNews API types ─────────────────────────────────────────────────────

type hnItem struct {
	ID          int    `json:"id"`
	Type        string `json:"type"`
	By          string `json:"by"`
	Time        int64  `json:"time"`
	Title       string `json:"title,omitempty"`
	URL         string `json:"url,omitempty"`
	Text        string `json:"text,omitempty"`
	Score       int    `json:"score,omitempty"`
	Descendants int    `json:"descendants,omitempty"`
	Kids        []int  `json:"kids,omitempty"`
}

type algoliaResponse struct {
	Hits []algoliaHit `json:"hits"`
}

type algoliaHit struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Author   string `json:"author"`
	Points   int    `json:"points"`
	ObjectID string `json:"objectID"`
	NumComments int `json:"num_comments"`
	CreatedAt string `json:"created_at"`
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func handleTopStories(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	count := 10
	if v, err := req.RequireInt("count"); err == nil && v > 0 {
		count = v
	}
	if count > 30 {
		count = 30
	}

	var ids []int
	if err := fetchJSON(ctx, firebaseBase+"/topstories.json", &ids); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch top stories: %v", err)), nil
	}

	if len(ids) > count {
		ids = ids[:count]
	}

	stories := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		var item hnItem
		if err := fetchJSON(ctx, fmt.Sprintf("%s/item/%d.json", firebaseBase, id), &item); err != nil {
			continue
		}
		stories = append(stories, map[string]any{
			"id":       item.ID,
			"title":    item.Title,
			"url":      item.URL,
			"score":    item.Score,
			"by":       item.By,
			"comments": item.Descendants,
		})
	}

	data, _ := json.MarshalIndent(stories, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleGetItem(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireInt("id")
	if err != nil {
		return mcp.NewToolResultError("id is required (integer)"), nil
	}

	var item hnItem
	if err := fetchJSON(ctx, fmt.Sprintf("%s/item/%d.json", firebaseBase, id), &item); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch item %d: %v", id, err)), nil
	}

	data, _ := json.MarshalIndent(item, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleGetUser(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	username, err := req.RequireString("username")
	if err != nil {
		return mcp.NewToolResultError("username is required"), nil
	}

	var user map[string]any
	if err := fetchJSON(ctx, fmt.Sprintf("%s/user/%s.json", firebaseBase, username), &user); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch user %q: %v", username, err)), nil
	}

	data, _ := json.MarshalIndent(user, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("query is required"), nil
	}

	count := 5
	if v, err := req.RequireInt("count"); err == nil && v > 0 {
		count = v
	}
	if count > 30 {
		count = 30
	}

	tags := "story"
	if v, err := req.RequireString("tags"); err == nil && v != "" {
		tags = v
	}

	url := fmt.Sprintf("%s/search?query=%s&tags=%s&hitsPerPage=%d",
		algoliaBase, query, tags, count)

	var result algoliaResponse
	if err := fetchJSON(ctx, url, &result); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Search failed: %v", err)), nil
	}

	stories := make([]map[string]any, 0, len(result.Hits))
	for _, hit := range result.Hits {
		stories = append(stories, map[string]any{
			"id":        hit.ObjectID,
			"title":     hit.Title,
			"url":       hit.URL,
			"author":    hit.Author,
			"points":    hit.Points,
			"comments":  hit.NumComments,
			"createdAt": hit.CreatedAt,
		})
	}

	data, _ := json.MarshalIndent(stories, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// ── HTTP helper ──────────────────────────────────────────────────────────────

func fetchJSON(ctx context.Context, url string, target any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}
