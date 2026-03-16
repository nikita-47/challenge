package main

import (
	"fmt"
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

// splitSentences splits text into individual sentences.
// Boundaries are detected at '.', '!', '?' followed by whitespace or end of string,
// and at '\n' line breaks.
func splitSentences(text string) []string {
	// Normalize line endings.
	text = strings.ReplaceAll(text, "\r\n", "\n")

	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	n := len(runes)

	for i := 0; i < n; i++ {
		ch := runes[i]

		if ch == '\n' {
			// Newline always ends the current sentence.
			s := strings.TrimSpace(current.String())
			if s != "" {
				sentences = append(sentences, s)
			}
			current.Reset()
			continue
		}

		current.WriteRune(ch)

		// Check for sentence-ending punctuation.
		if ch == '.' || ch == '!' || ch == '?' {
			// Look ahead: boundary if followed by whitespace, another sentence-ender, or end.
			isEnd := i+1 >= n
			isFollowedBySpace := i+1 < n && unicode.IsSpace(runes[i+1])

			if isEnd || isFollowedBySpace {
				s := strings.TrimSpace(current.String())
				if s != "" {
					sentences = append(sentences, s)
				}
				current.Reset()
			}
		}
	}

	// Flush any remaining text.
	s := strings.TrimSpace(current.String())
	if s != "" {
		sentences = append(sentences, s)
	}

	return sentences
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
