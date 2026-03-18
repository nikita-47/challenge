package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
)

// ragResultWithDoc pairs a SearchResult with the human-readable document name.
type ragResultWithDoc struct {
	SearchResult
	DocName string `json:"doc_name"`
}

// rewriteQuery calls Claude Haiku (non-streaming) to rewrite the user's question
// into a concise query optimized for semantic search against document chunks.
func rewriteQuery(apiKey, originalQuery string) (string, error) {
	systemPrompt := "You are a search query optimizer. Rewrite the user's question into a concise search query optimized for semantic search against document chunks. Output ONLY the rewritten query, nothing else."

	body, _ := json.Marshal(map[string]any{
		"model":      ModelHaiku,
		"max_tokens": 256,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": originalQuery},
		},
	})

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, errBody)
	}

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}

	var text strings.Builder
	for _, block := range result.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}

	return strings.TrimSpace(text.String()), nil
}

// filterResults splits results into passed (score >= threshold) and rejected.
// If threshold <= 0 all results are passed and rejected is empty.
func filterResults(results []SearchResult, threshold float64) (passed []SearchResult, rejected []SearchResult) {
	if threshold <= 0 {
		return results, []SearchResult{}
	}

	for _, r := range results {
		if r.Score >= threshold {
			passed = append(passed, r)
		} else {
			rejected = append(rejected, r)
		}
	}

	return passed, rejected
}

// ragResultWithDocFromSearchResult converts a SearchResult into ragResultWithDoc.
func ragResultWithDocFromSearchResult(sr SearchResult, docName string) ragResultWithDoc {
	return ragResultWithDoc{
		SearchResult: sr,
		DocName:      docName,
	}
}

// performRAGSearch orchestrates the full RAG pipeline:
// optional query rewrite → embed → search across docs → filter → inject XML context.
// Emits rag_step SSE events at each stage.
// Returns the effective message (XML context + original query) or "" if no results found.
func performRAGSearch(w http.ResponseWriter, apiKey string, docStore *DocumentStore, req chatRequest) (string, error) {
	ragStrategy := req.RagStrategy
	if ragStrategy == "" {
		ragStrategy = "auto"
	}

	ragTopK := req.RagTopK
	if ragTopK <= 0 {
		ragTopK = 5
	}

	originalQuery := req.Message
	searchQuery := originalQuery
	rewrittenQuery := ""

	// Step 1: optional query rewrite.
	sseWrite(w, map[string]any{
		"type":   "rag_step",
		"step":   "rewrite",
		"status": "running",
	})

	if req.RagQueryRewrite {
		rewritten, err := rewriteQuery(apiKey, originalQuery)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[rag] rewrite error: %v\n", err)
			sseWrite(w, map[string]any{
				"type":   "rag_step",
				"step":   "rewrite",
				"status": "error",
				"detail": map[string]any{"error": err.Error()},
			})
			// Non-fatal: fall back to original query.
		} else {
			rewrittenQuery = rewritten
			searchQuery = rewritten
			sseWrite(w, map[string]any{
				"type":   "rag_step",
				"step":   "rewrite",
				"status": "done",
				"detail": map[string]any{"original": originalQuery, "rewritten": rewrittenQuery},
			})
		}
	} else {
		sseWrite(w, map[string]any{
			"type":   "rag_step",
			"step":   "rewrite",
			"status": "skipped",
			"detail": map[string]any{"query": originalQuery},
		})
	}

	// Step 2: embed the search query.
	sseWrite(w, map[string]any{
		"type":   "rag_step",
		"step":   "embed",
		"status": "running",
	})

	queryEmbedding, embErr := GetEmbedding(searchQuery)
	if embErr != nil {
		sseWrite(w, map[string]any{
			"type":   "rag_step",
			"step":   "embed",
			"status": "error",
			"detail": map[string]any{"error": embErr.Error()},
		})
		return "", fmt.Errorf("embed query: %w", embErr)
	}

	sseWrite(w, map[string]any{
		"type":   "rag_step",
		"step":   "embed",
		"status": "done",
		"detail": map[string]any{"query": searchQuery},
	})

	// Step 3: search across all requested documents.
	sseWrite(w, map[string]any{
		"type":   "rag_step",
		"step":   "search",
		"status": "running",
	})

	var rawResults []SearchResult
	docNameByChunkID := make(map[string]string)

	for _, docID := range req.RagDocIDs {
		doc := docStore.Get(docID)
		if doc == nil || doc.IndexStatus != "ready" {
			continue
		}

		idx, loadErr := loadCombinedIndex(docStore.indexDir, docID)
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "[rag] failed to load index for %s: %v\n", docID, loadErr)
			continue
		}

		results := SearchChunks(idx, queryEmbedding, ragTopK, ragStrategy)
		for _, r := range results {
			rawResults = append(rawResults, r)
			docNameByChunkID[r.Chunk.ID] = doc.OriginalName
		}
	}

	// Sort all cross-doc results by score descending, keep top-K overall.
	sort.Slice(rawResults, func(i, j int) bool {
		return rawResults[i].Score > rawResults[j].Score
	})
	if ragTopK < len(rawResults) {
		rawResults = rawResults[:ragTopK]
	}

	sseWrite(w, map[string]any{
		"type":   "rag_step",
		"step":   "search",
		"status": "done",
		"detail": map[string]any{"total": len(rawResults)},
	})

	if len(rawResults) == 0 {
		return "", nil
	}

	// Step 4: threshold filter.
	sseWrite(w, map[string]any{
		"type":   "rag_step",
		"step":   "filter",
		"status": "running",
	})

	passed, rejected := filterResults(rawResults, req.RagThreshold)

	if req.RagThreshold > 0 {
		sseWrite(w, map[string]any{
			"type":   "rag_step",
			"step":   "filter",
			"status": "done",
			"detail": map[string]any{
				"passed":    len(passed),
				"rejected":  len(rejected),
				"threshold": req.RagThreshold,
			},
		})
	} else {
		sseWrite(w, map[string]any{
			"type":   "rag_step",
			"step":   "filter",
			"status": "skipped",
			"detail": map[string]any{"reason": "no threshold"},
		})
	}

	if len(passed) == 0 {
		return "", nil
	}

	// Step 5: build XML context block from passed chunks only.
	var xmlParts []string
	for _, r := range passed {
		docName := docNameByChunkID[r.Chunk.ID]
		xmlParts = append(xmlParts, fmt.Sprintf(
			"<document source=%q chunk=%q relevance=\"%.4f\">\n%s\n</document>",
			docName,
			r.Chunk.ID,
			r.Score,
			r.Chunk.Text,
		))
	}

	ragContext := "<documents>\n" +
		strings.Join(xmlParts, "\n") +
		"\n</documents>\n\nAnswer the user's question based on the document fragments above. Cite specific data, numbers, and identifiers from the documents. Respond in the same language as the user's question."

	sseWrite(w, map[string]any{
		"type":   "rag_step",
		"step":   "inject",
		"status": "done",
		"detail": map[string]any{"chunks": len(passed)},
	})

	// Build typed slices for the rag_context event.
	passedWithDoc := make([]ragResultWithDoc, 0, len(passed))
	for _, r := range passed {
		passedWithDoc = append(passedWithDoc, ragResultWithDocFromSearchResult(r, docNameByChunkID[r.Chunk.ID]))
	}

	allWithDoc := make([]ragResultWithDoc, 0, len(rawResults))
	for _, r := range rawResults {
		allWithDoc = append(allWithDoc, ragResultWithDocFromSearchResult(r, docNameByChunkID[r.Chunk.ID]))
	}

	rejectedWithDoc := make([]ragResultWithDoc, 0, len(rejected))
	for _, r := range rejected {
		rejectedWithDoc = append(rejectedWithDoc, ragResultWithDocFromSearchResult(r, docNameByChunkID[r.Chunk.ID]))
	}

	sseWrite(w, map[string]any{
		"type":            "rag_context",
		"results":         passedWithDoc,
		"all_results":     allWithDoc,
		"rejected":        rejectedWithDoc,
		"rewritten_query": rewrittenQuery,
		"threshold":       req.RagThreshold,
	})

	return ragContext + "\n\n" + originalQuery, nil
}
