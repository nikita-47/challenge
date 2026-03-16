package main

import (
	"math"
	"time"
)

// ChunkWithSimilarity is a chunk with cosine similarity to the next adjacent chunk.
// SimilarityToNext is nil for the last chunk in a strategy.
type ChunkWithSimilarity struct {
	Chunk
	SimilarityToNext *float64 `json:"similarity_to_next"`
	EmbeddingDim     int      `json:"embedding_dim"`
}

// StrategyResult contains chunks with similarity data and aggregate stats across a strategy.
type StrategyResult struct {
	Chunks        []ChunkWithSimilarity `json:"chunks"`
	AvgSimilarity float64               `json:"avg_similarity"`
	MinSimilarity float64               `json:"min_similarity"`
	MaxSimilarity float64               `json:"max_similarity"`
}

// ChunkResponse is the API response for the chunks endpoint.
// Raw embedding vectors are stripped; only similarity stats are returned.
type ChunkResponse struct {
	DocID          string         `json:"doc_id"`
	Filename       string         `json:"filename"`
	EmbeddingModel string         `json:"embedding_model"`
	CreatedAt      time.Time      `json:"created_at"`
	Size           StrategyResult `json:"size"`
	Sentence       StrategyResult `json:"sentence"`
	Structure      StrategyResult `json:"structure"`
	Semantic       StrategyResult `json:"semantic"`
}

// cosineSimilarity computes the cosine similarity between two equal-length vectors.
// Returns 0 if either vector has zero norm or they differ in length.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}

	return math.Round(dot/denom*10000) / 10000
}

// buildStrategyResult converts a slice of IndexedChunks into a StrategyResult,
// computing pairwise cosine similarities between consecutive chunks and aggregating stats.
func buildStrategyResult(chunks []IndexedChunk) StrategyResult {
	result := StrategyResult{
		Chunks: make([]ChunkWithSimilarity, 0, len(chunks)),
	}

	if len(chunks) == 0 {
		return result
	}

	similarities := make([]float64, 0, len(chunks)-1)

	for i, ic := range chunks {
		cwsim := ChunkWithSimilarity{
			Chunk:        ic.Chunk,
			EmbeddingDim: len(ic.Embedding),
		}

		if i < len(chunks)-1 {
			sim := cosineSimilarity(ic.Embedding, chunks[i+1].Embedding)
			cwsim.SimilarityToNext = &sim
			similarities = append(similarities, sim)
		}

		result.Chunks = append(result.Chunks, cwsim)
	}

	if len(similarities) == 0 {
		// Single chunk: leave avg/min/max at zero.
		return result
	}

	sum := 0.0
	minSim := similarities[0]
	maxSim := similarities[0]

	for _, s := range similarities {
		sum += s
		if s < minSim {
			minSim = s
		}
		if s > maxSim {
			maxSim = s
		}
	}

	result.AvgSimilarity = math.Round(sum/float64(len(similarities))*10000) / 10000
	result.MinSimilarity = minSim
	result.MaxSimilarity = maxSim

	return result
}

// buildChunkResponse transforms a CombinedIndex into a ChunkResponse,
// stripping raw embeddings and adding per-chunk and aggregate similarity stats.
func buildChunkResponse(idx *CombinedIndex) *ChunkResponse {
	return &ChunkResponse{
		DocID:          idx.DocID,
		Filename:       idx.Filename,
		EmbeddingModel: idx.EmbeddingModel,
		CreatedAt:      idx.CreatedAt,
		Size:           buildStrategyResult(idx.Size),
		Sentence:       buildStrategyResult(idx.Sentence),
		Structure:      buildStrategyResult(idx.Structure),
		Semantic:       buildStrategyResult(idx.Semantic),
	}
}
