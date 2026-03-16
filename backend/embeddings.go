package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const ollamaEmbeddingsURL = "http://localhost:11434/api/embeddings"
const ollamaEmbeddingModel = "nomic-embed-text"

var ollamaClient = &http.Client{
	Timeout: 30 * time.Second,
}

// GetEmbedding calls Ollama to generate an embedding vector for the given text.
func GetEmbedding(text string) ([]float64, error) {
	reqBody, err := json.Marshal(map[string]string{
		"model":  ollamaEmbeddingModel,
		"prompt": text,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	resp, err := ollamaClient.Post(ollamaEmbeddingsURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			return nil, fmt.Errorf("Ollama not reachable: %w", err)
		}
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("Ollama returned empty embedding")
	}

	return result.Embedding, nil
}

// GetEmbeddings generates embeddings for a slice of texts sequentially.
func GetEmbeddings(texts []string) ([][]float64, error) {
	embeddings := make([][]float64, 0, len(texts))
	for i, text := range texts {
		emb, err := GetEmbedding(text)
		if err != nil {
			return nil, fmt.Errorf("embedding %d failed: %w", i, err)
		}
		embeddings = append(embeddings, emb)
	}
	return embeddings, nil
}
