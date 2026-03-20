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
		sseWrite(w, map[string]any{
			"type":   "rag_step",
			"step":   "filter",
			"status": "skipped",
			"detail": map[string]any{"reason": "no results to filter"},
		})
		sseWrite(w, map[string]any{
			"type":   "rag_step",
			"step":   "inject",
			"status": "skipped",
			"detail": map[string]any{"reason": "no results found"},
		})
		sseWrite(w, map[string]any{
			"type":            "rag_context",
			"results":         []ragResultWithDoc{},
			"all_results":     []ragResultWithDoc{},
			"rejected":        []ragResultWithDoc{},
			"rewritten_query": rewrittenQuery,
			"threshold":       req.RagThreshold,
			"no_context":      true,
		})
		noResultsMsg := "<rag_no_context>\nThe user's question was searched against the document database, but no matching chunks were found at all.\nYou MUST respond that you cannot answer this question based on the available documents.\nSuggest the user: try different search terms, check that the right documents are selected, or upload more relevant documents.\nDo NOT attempt to answer from your general knowledge.\n</rag_no_context>\n\n" + originalQuery
		return noResultsMsg, nil
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
		rejectedWithDoc := make([]ragResultWithDoc, 0, len(rejected))
		for _, r := range rejected {
			rejectedWithDoc = append(rejectedWithDoc, ragResultWithDocFromSearchResult(r, docNameByChunkID[r.Chunk.ID]))
		}
		allWithDoc := make([]ragResultWithDoc, 0, len(rawResults))
		for _, r := range rawResults {
			allWithDoc = append(allWithDoc, ragResultWithDocFromSearchResult(r, docNameByChunkID[r.Chunk.ID]))
		}
		sseWrite(w, map[string]any{
			"type":   "rag_step",
			"step":   "inject",
			"status": "skipped",
			"detail": map[string]any{"reason": "all chunks below threshold"},
		})
		sseWrite(w, map[string]any{
			"type":            "rag_context",
			"results":         []ragResultWithDoc{},
			"all_results":     allWithDoc,
			"rejected":        rejectedWithDoc,
			"rewritten_query": rewrittenQuery,
			"threshold":       req.RagThreshold,
			"no_context":      true,
		})
		noContextMsg := "<rag_no_context>\nThe user's question was searched against the document database, but no chunks met the relevance threshold (" + fmt.Sprintf("%.2f", req.RagThreshold) + ").\nYou MUST respond that you cannot answer this question based on the available documents.\nSuggest the user: lower the relevance threshold, rephrase the question, or upload more relevant documents.\nDo NOT attempt to answer from your general knowledge.\n</rag_no_context>\n\n" + originalQuery
		return noContextMsg, nil
	}

	// Step 5: build XML context block from passed chunks only.
	// Each document tag gets a numeric ref attribute starting from 1.
	var xmlParts []string
	for i, r := range passed {
		docName := docNameByChunkID[r.Chunk.ID]
		xmlParts = append(xmlParts, fmt.Sprintf(
			"<document ref=\"%d\" source=%q chunk=%q relevance=\"%.4f\">\n%s\n</document>",
			i+1,
			docName,
			r.Chunk.ID,
			r.Score,
			r.Chunk.Text,
		))
	}

	citationRules := `CITATION RULES:
- Base your answer ONLY on the document fragments above.
- When you use information from a document, add an inline citation using the format [N] where N is the document ref number.
- At the very end of your response, add a sources block in this exact format:
<!-- sources
[{"ref":1,"source":"filename.pdf","chunk":"chunk_id","score":0.82},{"ref":2,...}]
-->
- Only include sources you actually cited in your answer.
- If the documents do not contain enough information to answer the question, say explicitly that you cannot answer based on the available documents and suggest how the user might refine their query.
- Do NOT invent or hallucinate information not present in the provided documents.
- Include direct quotes from the documents where appropriate, using blockquotes (> quote text).
- Respond in the same language as the user's question.`

	ragContext := "<documents>\n" +
		strings.Join(xmlParts, "\n") +
		"\n</documents>\n\n" + citationRules

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
