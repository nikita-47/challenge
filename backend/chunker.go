package main

import (
	"fmt"
	"math"
	"strings"
	"unicode"
)

// Chunk represents a piece of text extracted from a document.
type Chunk struct {
	ID       string            `json:"id"`
	DocID    string            `json:"doc_id"`
	Text     string            `json:"text"`
	Index    int               `json:"index"`
	Metadata map[string]string `json:"metadata"`
}

// ChunkSize splits text using a sliding window over runes (unicode-safe).
// step = chunkSize - overlap.
func ChunkSize(text string, docID string, filename string, chunkSize int, overlap int) []Chunk {
	runes := []rune(text)
	total := len(runes)
	if total == 0 {
		return []Chunk{}
	}

	step := chunkSize - overlap
	if step <= 0 {
		step = chunkSize
	}

	var chunks []Chunk
	idx := 0
	pos := 0

	for pos < total {
		end := pos + chunkSize
		if end > total {
			end = total
		}

		chunkText := strings.TrimSpace(string(runes[pos:end]))
		if chunkText != "" {
			chunkID := fmt.Sprintf("%s_chunk_%d", docID, idx)
			chunks = append(chunks, Chunk{
				ID:    chunkID,
				DocID: docID,
				Text:  chunkText,
				Index: idx,
				Metadata: map[string]string{
					"source":   filename,
					"title":    filename,
					"section":  fmt.Sprintf("chunk_%d", idx),
					"chunk_id": chunkID,
				},
			})
			idx++
		}

		if end == total {
			break
		}
		pos += step
	}

	return chunks
}

// ChunkSentence splits text into sentences and groups them into chunks.
// Sentence boundaries are detected at '.', '!', '?' followed by whitespace or end of
// string, and at newlines. sentencesPerChunk sentences form one chunk; the last
// `overlap` sentences of the previous chunk are prepended to the next chunk.
func ChunkSentence(text string, docID string, filename string, sentencesPerChunk int, overlap int) []Chunk {
	if sentencesPerChunk <= 0 {
		sentencesPerChunk = 5
	}
	if overlap < 0 {
		overlap = 0
	}

	sentences := splitSentences(text)

	var chunks []Chunk
	chunkIdx := 0
	total := len(sentences)

	step := sentencesPerChunk - overlap
	if step <= 0 {
		step = sentencesPerChunk
	}

	for start := 0; start < total; start += step {
		end := start + sentencesPerChunk
		if end > total {
			end = total
		}

		chunkText := strings.TrimSpace(strings.Join(sentences[start:end], " "))
		if chunkText == "" {
			if end >= total {
				break
			}
			continue
		}

		chunkID := fmt.Sprintf("%s_chunk_%d", docID, chunkIdx)
		section := fmt.Sprintf("sentences_%d-%d", start, end-1)
		chunks = append(chunks, Chunk{
			ID:    chunkID,
			DocID: docID,
			Text:  chunkText,
			Index: chunkIdx,
			Metadata: map[string]string{
				"source":   filename,
				"title":    filename,
				"section":  section,
				"chunk_id": chunkID,
			},
		})
		chunkIdx++

		if end >= total {
			break
		}
	}

	return chunks
}

// sentenceSpan holds a sentence's text and its byte offsets in the original string.
type sentenceSpan struct {
	text  string
	start int // byte offset in original (normalized) text
	end   int // byte offset (exclusive)
}

// splitSentences splits text into individual sentences.
// Boundaries are detected at '.', '!', '?' followed by whitespace or end of string,
// and at '\n' line breaks.
func splitSentences(text string) []string {
	spans := splitSentenceSpans(text)
	out := make([]string, len(spans))
	for i, sp := range spans {
		out[i] = sp.text
	}
	return out
}

// splitSentenceSpans splits text into sentences and tracks their byte positions.
func splitSentenceSpans(text string) []sentenceSpan {
	// Normalize line endings.
	text = strings.ReplaceAll(text, "\r\n", "\n")

	var spans []sentenceSpan
	var current strings.Builder

	runes := []rune(text)
	n := len(runes)

	// Track byte offset of current sentence start.
	bytePos := 0     // current byte position in normalized text
	sentStart := 0    // byte offset where current sentence content starts
	sentStartSet := false

	for i := 0; i < n; i++ {
		ch := runes[i]
		charLen := len(string(ch))

		if ch == '\n' {
			s := strings.TrimSpace(current.String())
			if s != "" {
				spans = append(spans, sentenceSpan{
					text:  s,
					start: sentStart,
					end:   bytePos + charLen,
				})
			}
			current.Reset()
			sentStartSet = false
			bytePos += charLen
			continue
		}

		if !sentStartSet && !unicode.IsSpace(ch) {
			sentStart = bytePos
			sentStartSet = true
		}

		current.WriteRune(ch)

		if ch == '.' || ch == '!' || ch == '?' {
			isEnd := i+1 >= n
			isFollowedBySpace := i+1 < n && unicode.IsSpace(runes[i+1])

			if isEnd || isFollowedBySpace {
				s := strings.TrimSpace(current.String())
				if s != "" {
					spans = append(spans, sentenceSpan{
						text:  s,
						start: sentStart,
						end:   bytePos + charLen,
					})
				}
				current.Reset()
				sentStartSet = false
			}
		}

		bytePos += charLen
	}

	s := strings.TrimSpace(current.String())
	if s != "" {
		spans = append(spans, sentenceSpan{
			text:  s,
			start: sentStart,
			end:   bytePos,
		})
	}

	return spans
}

// ChunkSemantic splits text into semantically coherent chunks by detecting
// topic boundaries via cosine similarity drops between consecutive sentence embeddings.
//
// Algorithm:
//  1. Split text into sentences via splitSentences.
//  2. Embed each sentence with GetEmbedding.
//  3. Compute cosine similarity between every pair of adjacent sentence embeddings.
//  4. A boundary is placed where similarity < (mean − 1.0 × stddev), meaning a
//     significant topic shift. Minimum chunk size: 2 sentences. Maximum: 15 sentences
//     (oversized chunks are split further).
//  5. If fewer than 3 sentences exist, the entire text is returned as one chunk.
func ChunkSemantic(text, docID, filename string) []Chunk {
	// Normalize line endings consistently (splitSentenceSpans does it too, but
	// we need the same normalized text for substring extraction).
	normalizedText := strings.ReplaceAll(text, "\r\n", "\n")
	spans := splitSentenceSpans(normalizedText)

	if len(spans) < 3 {
		if len(spans) == 0 {
			return []Chunk{}
		}
		chunkID := fmt.Sprintf("%s_chunk_0", docID)
		return []Chunk{{
			ID:    chunkID,
			DocID: docID,
			Text:  strings.TrimSpace(normalizedText),
			Index: 0,
			Metadata: map[string]string{
				"source":   filename,
				"title":    filename,
				"section":  "semantic_0",
				"chunk_id": chunkID,
			},
		}}
	}

	// Embed all sentences.
	embeddings := make([][]float64, 0, len(spans))
	for _, sp := range spans {
		emb, err := GetEmbedding(sp.text)
		if err != nil {
			chunkID := fmt.Sprintf("%s_chunk_0", docID)
			return []Chunk{{
				ID:    chunkID,
				DocID: docID,
				Text:  strings.TrimSpace(normalizedText),
				Index: 0,
				Metadata: map[string]string{
					"source":   filename,
					"title":    filename,
					"section":  "semantic_0",
					"chunk_id": chunkID,
				},
			}}
		}
		embeddings = append(embeddings, emb)
	}

	// Compute similarities between consecutive sentences.
	sims := make([]float64, len(spans)-1)
	for i := 0; i < len(spans)-1; i++ {
		sims[i] = cosineSimilarity(embeddings[i], embeddings[i+1])
	}

	// Compute mean and stddev of similarities.
	var sum float64
	for _, s := range sims {
		sum += s
	}
	mean := sum / float64(len(sims))

	var variance float64
	for _, s := range sims {
		diff := s - mean
		variance += diff * diff
	}
	stddev := math.Sqrt(variance / float64(len(sims)))
	threshold := mean - 1.0*stddev

	// Identify boundary indices (sentence index where a new chunk starts).
	const maxSentences = 15
	type spanGroup struct {
		startIdx int // index in spans slice
		endIdx   int // exclusive
	}
	var groups []spanGroup
	start := 0

	for i := 0; i < len(sims); i++ {
		chunkLen := i - start + 1
		nextChunkLen := i + 1 - start + 1

		isBoundary := sims[i] < threshold
		isOversize := nextChunkLen > maxSentences

		if (isBoundary || isOversize) && chunkLen >= 2 {
			groups = append(groups, spanGroup{startIdx: start, endIdx: i + 1})
			start = i + 1
		}
	}
	if start < len(spans) {
		groups = append(groups, spanGroup{startIdx: start, endIdx: len(spans)})
	}

	// Build chunks by extracting original text between span boundaries.
	var chunks []Chunk
	for i, g := range groups {
		byteStart := spans[g.startIdx].start
		byteEnd := spans[g.endIdx-1].end
		chunkText := strings.TrimSpace(normalizedText[byteStart:byteEnd])
		if chunkText == "" {
			continue
		}

		chunkID := fmt.Sprintf("%s_chunk_%d", docID, i)
		chunks = append(chunks, Chunk{
			ID:    chunkID,
			DocID: docID,
			Text:  chunkText,
			Index: i,
			Metadata: map[string]string{
				"source":   filename,
				"title":    filename,
				"section":  fmt.Sprintf("semantic_%d", i),
				"chunk_id": chunkID,
			},
		})
	}
	return chunks
}

// ChunkStructure dispatches to the appropriate structure-aware chunker
// based on content type.
func ChunkStructure(text, docID, filename, contentType string) []Chunk {
	switch contentType {
	case "text/markdown":
		return chunkMarkdown(text, docID, filename)
	case "application/pdf":
		// PDF structure chunking requires pages — caller should use chunkPDF directly.
		return chunkPlaintext(text, docID, filename)
	default:
		return chunkPlaintext(text, docID, filename)
	}
}

// chunkMarkdown splits text on ## and # headings.
// Each section becomes one chunk; oversized sections are split further with fixed chunking.
func chunkMarkdown(text, docID, filename string) []Chunk {
	// Normalize line endings.
	text = strings.ReplaceAll(text, "\r\n", "\n")

	lines := strings.Split(text, "\n")
	var sections []struct {
		heading string
		body    strings.Builder
	}

	current := struct {
		heading string
		body    strings.Builder
	}{heading: ""}

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "# ") {
			// Save current section if non-empty.
			if strings.TrimSpace(current.body.String()) != "" || current.heading != "" {
				sections = append(sections, current)
			}
			current = struct {
				heading string
				body    strings.Builder
			}{heading: strings.TrimPrefix(strings.TrimPrefix(line, "## "), "# ")}
			current.body.WriteString(line + "\n")
		} else {
			current.body.WriteString(line + "\n")
		}
	}
	// Flush last section.
	if strings.TrimSpace(current.body.String()) != "" || current.heading != "" {
		sections = append(sections, current)
	}

	var chunks []Chunk
	idx := 0

	for _, sec := range sections {
		sectionText := strings.TrimSpace(sec.body.String())
		if sectionText == "" {
			continue
		}

		heading := sec.heading
		if heading == "" {
			heading = "intro"
		}

		// If section is too large, split further with size chunking at 2000 runes.
		if len([]rune(sectionText)) > 2000 {
			sub := ChunkSize(sectionText, docID, filename, 2000, 0)
			for _, sc := range sub {
				chunkID := fmt.Sprintf("%s_chunk_%d", docID, idx)
				chunks = append(chunks, Chunk{
					ID:    chunkID,
					DocID: docID,
					Text:  sc.Text,
					Index: idx,
					Metadata: map[string]string{
						"source":   filename,
						"title":    filename,
						"section":  heading,
						"chunk_id": chunkID,
					},
				})
				idx++
			}
		} else {
			chunkID := fmt.Sprintf("%s_chunk_%d", docID, idx)
			chunks = append(chunks, Chunk{
				ID:    chunkID,
				DocID: docID,
				Text:  sectionText,
				Index: idx,
				Metadata: map[string]string{
					"source":   filename,
					"title":    filename,
					"section":  heading,
					"chunk_id": chunkID,
				},
			})
			idx++
		}
	}

	return chunks
}

// chunkPDF creates one chunk per page.
func chunkPDF(pages []string, docID, filename string) []Chunk {
	var chunks []Chunk
	idx := 0

	for i, pageText := range pages {
		pageText = strings.TrimSpace(pageText)
		if pageText == "" {
			continue
		}

		chunkID := fmt.Sprintf("%s_chunk_%d", docID, idx)
		chunks = append(chunks, Chunk{
			ID:    chunkID,
			DocID: docID,
			Text:  pageText,
			Index: idx,
			Metadata: map[string]string{
				"source":   filename,
				"title":    filename,
				"section":  fmt.Sprintf("page_%d", i+1),
				"chunk_id": chunkID,
			},
		})
		idx++
	}

	return chunks
}

// chunkPlaintext splits text on double newlines (paragraph boundaries).
func chunkPlaintext(text, docID, filename string) []Chunk {
	// Normalize line endings.
	text = strings.ReplaceAll(text, "\r\n", "\n")
	paragraphs := strings.Split(text, "\n\n")

	var chunks []Chunk
	idx := 0

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		chunkID := fmt.Sprintf("%s_chunk_%d", docID, idx)
		chunks = append(chunks, Chunk{
			ID:    chunkID,
			DocID: docID,
			Text:  para,
			Index: idx,
			Metadata: map[string]string{
				"source":   filename,
				"title":    filename,
				"section":  fmt.Sprintf("para_%d", idx),
				"chunk_id": chunkID,
			},
		})
		idx++
	}

	return chunks
}
