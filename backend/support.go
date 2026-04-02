package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

type supportChatRequest struct {
	Message  string `json:"message"`
	TicketID string `json:"ticketId,omitempty"`
	History  []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"history"`
}

func handleSupportChat(w http.ResponseWriter, r *http.Request, apiKey string, mcpMgr *MCPManager, docStore *DocumentStore) {
	var req supportChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	sseSetup(w)

	// Find FAQ doc IDs: only docs whose OriginalName contains "faq-".
	var faqDocIDs []string
	for _, doc := range docStore.List() {
		if strings.Contains(doc.OriginalName, "faq-") && doc.IndexStatus == "ready" {
			faqDocIDs = append(faqDocIDs, doc.ID)
		}
	}

	// Attempt to fetch ticket context via MCP if ticketId is provided.
	ticketJSON := ""
	if req.TicketID != "" && mcpMgr != nil {
		result, err := mcpMgr.CallTool(context.Background(), "tickets", "ticket_get", map[string]any{
			"id": req.TicketID,
		})
		if err == nil && result != nil {
			for _, c := range result.Content {
				if tc, ok := c.(mcp.TextContent); ok {
					ticketJSON = tc.Text
					break
				}
			}
		}
		// If CallTool fails, we skip gracefully (ticketJSON remains "").
	}

	// Build RAG-enriched message if FAQ docs are available.
	effectiveMessage := req.Message
	if len(faqDocIDs) > 0 && req.Message != "" {
		ragReq := chatRequest{
			Message:         req.Message,
			RagDocIDs:       faqDocIDs,
			RagQueryRewrite: true,
			RagTopK:         5,
			RagThreshold:    0.2,
			RagStrategy:     "auto",
		}
		enriched, ragErr := performRAGSearch(w, apiKey, docStore, ragReq)
		if ragErr == nil && enriched != "" {
			effectiveMessage = enriched
		}
	}

	// Build system prompt.
	systemPrompt := "You are a friendly and helpful support assistant for the AI Challenge chat application.\n" +
		"Answer user questions based on the provided FAQ and documentation context.\n" +
		"If a support ticket is referenced, consider its history and context when answering.\n" +
		"If you cannot find the answer in the documentation, suggest creating a support ticket for further help.\n" +
		"Be concise and helpful. Always respond in the same language as the user's message.\n" +
		"Do not make up information. If unsure, say so clearly."

	if ticketJSON != "" {
		systemPrompt += "\n\nCurrent ticket context:\n<ticket>\n" + ticketJSON + "\n</ticket>"
	}

	// Build messages array from history (last 10 turns) + current message.
	history := req.History
	if len(history) > 10 {
		history = history[len(history)-10:]
	}

	messages := make([]map[string]string, 0, len(history)+1)
	for _, h := range history {
		messages = append(messages, map[string]string{
			"role":    h.Role,
			"content": h.Content,
		})
	}
	messages = append(messages, map[string]string{
		"role":    "user",
		"content": effectiveMessage,
	})

	// Build and send the Claude API request.
	reqBody, _ := json.Marshal(map[string]any{
		"model":       ModelHaiku,
		"max_tokens":  2048,
		"stream":      true,
		"temperature": 0.3,
		"system":      systemPrompt,
		"messages":    messages,
	})

	httpReq, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		sseWrite(w, map[string]any{"type": "error", "message": err.Error()})
		sseWrite(w, map[string]any{"type": "done"})
		return
	}
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		sseWrite(w, map[string]any{"type": "error", "message": err.Error()})
		sseWrite(w, map[string]any{"type": "done"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		sseWrite(w, map[string]any{
			"type":    "error",
			"message": fmt.Sprintf("Claude API error: %d", resp.StatusCode),
		})
		sseWrite(w, map[string]any{"type": "done"})
		return
	}

	_, _, streamErr := readStream(resp.Body, func(token string) {
		sseWrite(w, map[string]any{"type": "text_delta", "text": token})
	})
	if streamErr != nil {
		sseWrite(w, map[string]any{"type": "error", "message": streamErr.Error()})
		sseWrite(w, map[string]any{"type": "done"})
		return
	}

	sseWrite(w, map[string]any{"type": "done"})
}
