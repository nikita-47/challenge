package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CombinedIndex is the persisted index for a document, containing all 4 chunking strategies.
type CombinedIndex struct {
	DocID          string         `json:"doc_id"`
	Filename       string         `json:"filename"`
	EmbeddingModel string         `json:"embedding_model"`
	CreatedAt      time.Time      `json:"created_at"`
	Size           []IndexedChunk `json:"size"`
	Sentence       []IndexedChunk `json:"sentence"`
	Structure      []IndexedChunk `json:"structure"`
	Semantic       []IndexedChunk `json:"semantic"`
}

// IndexedChunk extends Chunk with a pre-computed embedding vector.
type IndexedChunk struct {
	Chunk
	Embedding []float64 `json:"embedding"`
}

// runIndexPipeline asynchronously indexes a document using all 4 chunking strategies.
func runIndexPipeline(store *DocumentStore, doc *DocumentMeta, filePath string, chunkSize, overlap int) {
	doc.IndexStatus = "indexing"
	store.Update(doc)

	indexPath := filepath.Join(store.indexDir, doc.ID+".json")

	if err := doIndex(store, doc, filePath, indexPath, chunkSize, overlap); err != nil {
		log.Printf("[indexer] error indexing %s: %v", doc.ID, err)
		doc.IndexStatus = "error"
		doc.IndexError = err.Error()
		store.Update(doc)
		return
	}

	doc.IndexStatus = "ready"
	doc.IndexError = ""
	store.Update(doc)
}

// doIndex performs the actual indexing work across all 4 strategies and returns any error.
func doIndex(store *DocumentStore, doc *DocumentMeta, filePath string, indexPath string, chunkSize, overlap int) error {
	var sizeChunks, sentenceChunks, structureChunks, semanticChunks []Chunk

	if doc.ContentType == "application/pdf" {
		pages, err := ExtractPDFText(filePath)
		if err != nil {
			return fmt.Errorf("PDF extraction failed: %w", err)
		}
		combined := strings.Join(pages, "\n\n")

		sizeChunks = ChunkSize(combined, doc.ID, doc.OriginalName, chunkSize, overlap)
		sentenceChunks = ChunkSentence(combined, doc.ID, doc.OriginalName, 5, 1)
		structureChunks = chunkPDF(pages, doc.ID, doc.OriginalName)
		semanticChunks = ChunkSemantic(combined, doc.ID, doc.OriginalName)
	} else {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		text := string(data)

		sizeChunks = ChunkSize(text, doc.ID, doc.OriginalName, chunkSize, overlap)
		sentenceChunks = ChunkSentence(text, doc.ID, doc.OriginalName, 5, 1)
		structureChunks = ChunkStructure(text, doc.ID, doc.OriginalName, doc.ContentType)
		semanticChunks = ChunkSemantic(text, doc.ID, doc.OriginalName)
	}

	indexedSize, err := embedChunks(sizeChunks)
	if err != nil {
		return fmt.Errorf("embedding size chunks failed: %w", err)
	}

	indexedSentence, err := embedChunks(sentenceChunks)
	if err != nil {
		return fmt.Errorf("embedding sentence chunks failed: %w", err)
	}

	indexedStructure, err := embedChunks(structureChunks)
	if err != nil {
		return fmt.Errorf("embedding structure chunks failed: %w", err)
	}

	indexedSemantic, err := embedChunks(semanticChunks)
	if err != nil {
		return fmt.Errorf("embedding semantic chunks failed: %w", err)
	}

	idx := CombinedIndex{
		DocID:          doc.ID,
		Filename:       doc.OriginalName,
		EmbeddingModel: ollamaEmbeddingModel,
		CreatedAt:      time.Now(),
		Size:           indexedSize,
		Sentence:       indexedSentence,
		Structure:      indexedStructure,
		Semantic:       indexedSemantic,
	}

	jsonData, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal combined index: %w", err)
	}

	if err := os.WriteFile(indexPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	doc.ChunkCount = len(sizeChunks) + len(sentenceChunks) + len(structureChunks) + len(semanticChunks)
	doc.ChunkStrategy = "all"
	doc.EmbeddingModel = ollamaEmbeddingModel

	return nil
}

// embedChunks generates embeddings for a slice of chunks and returns IndexedChunk slice.
func embedChunks(chunks []Chunk) ([]IndexedChunk, error) {
	indexed := make([]IndexedChunk, 0, len(chunks))
	for _, chunk := range chunks {
		emb, err := GetEmbedding(chunk.Text)
		if err != nil {
			return nil, fmt.Errorf("embedding chunk %d failed: %w", chunk.Index, err)
		}
		indexed = append(indexed, IndexedChunk{
			Chunk:     chunk,
			Embedding: emb,
		})
	}
	return indexed, nil
}
