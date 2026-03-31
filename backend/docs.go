package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DocumentMeta holds metadata for an uploaded and indexed document.
type DocumentMeta struct {
	ID              string    `json:"id"`
	Filename        string    `json:"filename"`
	OriginalName    string    `json:"original_name"`
	ContentType     string    `json:"content_type"`
	Size            int64     `json:"size"`
	UploadedAt      time.Time `json:"uploaded_at"`
	ChunkCount      int       `json:"chunk_count"`
	ChunkStrategy   string    `json:"chunk_strategy"`
	IndexStatus     string    `json:"index_status"`
	IndexError      string    `json:"index_error,omitempty"`
	EmbeddingModel  string    `json:"embedding_model"`
	ChunkSizeParam  int       `json:"chunk_size_param"`
	OverlapParam    int       `json:"overlap_param"`
}

// DocumentStore manages documents with thread-safe access and JSON persistence.
type DocumentStore struct {
	mu        sync.RWMutex
	docs      map[string]*DocumentMeta
	metaPath  string
	uploadDir string
	indexDir  string
	dirty     bool
	saveTimer *time.Timer
}

// NewDocumentStore creates a new DocumentStore and ensures storage directories exist.
func NewDocumentStore(metaPath, uploadDir, indexDir string) *DocumentStore {
	_ = os.MkdirAll(filepath.Dir(metaPath), 0755)
	_ = os.MkdirAll(uploadDir, 0755)
	_ = os.MkdirAll(indexDir, 0755)
	return &DocumentStore{
		docs:      make(map[string]*DocumentMeta),
		metaPath:  metaPath,
		uploadDir: uploadDir,
		indexDir:  indexDir,
	}
}

// Load reads document metadata from the JSON file into memory.
func (ds *DocumentStore) Load() error {
	data, err := os.ReadFile(ds.metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var docs []*DocumentMeta
	if err := json.Unmarshal(data, &docs); err != nil {
		return err
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	for _, d := range docs {
		ds.docs[d.ID] = d
	}
	return nil
}

// Save writes all documents to the JSON file.
func (ds *DocumentStore) Save() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	docs := make([]*DocumentMeta, 0, len(ds.docs))
	for _, d := range ds.docs {
		docs = append(docs, d)
	}
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].UploadedAt.Before(docs[j].UploadedAt)
	})

	data, err := json.MarshalIndent(docs, "", "  ")
	if err != nil {
		return err
	}

	ds.dirty = false
	return os.WriteFile(ds.metaPath, data, 0644)
}

// debouncedSave marks dirty and schedules a save after 2 seconds.
// Must be called with the write lock held.
func (ds *DocumentStore) debouncedSave() {
	ds.dirty = true
	if ds.saveTimer != nil {
		ds.saveTimer.Stop()
	}
	ds.saveTimer = time.AfterFunc(2*time.Second, func() {
		_ = ds.Save()
	})
}

// Add inserts a document into the store and persists.
func (ds *DocumentStore) Add(doc *DocumentMeta) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.docs[doc.ID] = doc
	ds.debouncedSave()
}

// Get retrieves a document by ID.
func (ds *DocumentStore) Get(id string) *DocumentMeta {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	return ds.docs[id]
}

// List returns all documents sorted by upload time (ascending).
func (ds *DocumentStore) List() []*DocumentMeta {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	docs := make([]*DocumentMeta, 0, len(ds.docs))
	for _, d := range ds.docs {
		docs = append(docs, d)
	}
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].UploadedAt.Before(docs[j].UploadedAt)
	})
	return docs
}

// Update replaces the stored document metadata and persists.
func (ds *DocumentStore) Update(doc *DocumentMeta) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.docs[doc.ID] = doc
	ds.debouncedSave()
}

// Delete removes a document by ID and persists. Returns error if not found.
func (ds *DocumentStore) Delete(id string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if _, ok := ds.docs[id]; !ok {
		return fmt.Errorf("document not found: %s", id)
	}

	delete(ds.docs, id)
	ds.debouncedSave()
	return nil
}

// generateDocID returns a random 8-character hex string.
func generateDocID() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// parseIntParam parses a string as an int, returning defaultVal if empty or invalid.
// The result is clamped to [min, max].
func parseIntParam(s string, defaultVal, min, max int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// ─── HTTP handlers ────────────────────────────────────────────────────────────

func handleListDocs(w http.ResponseWriter, r *http.Request, store *DocumentStore) {
	docs := store.List()
	if docs == nil {
		docs = []*DocumentMeta{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docs)
}

func handleUploadDoc(w http.ResponseWriter, r *http.Request, store *DocumentStore) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file field: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	originalName := header.Filename
	ext := strings.ToLower(filepath.Ext(originalName))
	if ext != ".txt" && ext != ".md" && ext != ".pdf" {
		http.Error(w, "unsupported file type: only .txt, .md, .pdf are allowed", http.StatusBadRequest)
		return
	}

	id, err := generateDocID()
	if err != nil {
		http.Error(w, "failed to generate ID: "+err.Error(), http.StatusInternalServerError)
		return
	}

	storedFilename := id + "_" + originalName
	destPath := filepath.Join(store.uploadDir, storedFilename)

	destFile, err := os.Create(destPath)
	if err != nil {
		http.Error(w, "failed to save file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer destFile.Close()

	size, err := io.Copy(destFile, file)
	if err != nil {
		http.Error(w, "failed to write file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	contentType := "text/plain"
	switch ext {
	case ".md":
		contentType = "text/markdown"
	case ".pdf":
		contentType = "application/pdf"
	}

	chunkSize := parseIntParam(r.FormValue("chunk_size"), 1000, 100, 5000)
	overlap := parseIntParam(r.FormValue("overlap"), 200, 0, chunkSize/2)

	doc := &DocumentMeta{
		ID:             id,
		Filename:       storedFilename,
		OriginalName:   originalName,
		ContentType:    contentType,
		Size:           size,
		UploadedAt:     time.Now(),
		ChunkStrategy:  "all",
		IndexStatus:    "pending",
		EmbeddingModel: "nomic-embed-text",
		ChunkSizeParam: chunkSize,
		OverlapParam:   overlap,
	}
	store.Add(doc)

	go runIndexPipeline(store, doc, destPath, chunkSize, overlap)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(doc)
}

func handleGetDoc(w http.ResponseWriter, r *http.Request, store *DocumentStore, id string) {
	doc := store.Get(id)
	if doc == nil {
		http.Error(w, "document not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func handleDeleteDoc(w http.ResponseWriter, r *http.Request, store *DocumentStore, id string) {
	doc := store.Get(id)
	if doc == nil {
		http.Error(w, "document not found", http.StatusNotFound)
		return
	}

	if doc.IndexStatus == "indexing" {
		http.Error(w, "cannot delete document while indexing is in progress", http.StatusConflict)
		return
	}

	// Delete upload file.
	uploadPath := filepath.Join(store.uploadDir, doc.Filename)
	if err := os.Remove(uploadPath); err != nil && !os.IsNotExist(err) {
		http.Error(w, "failed to delete upload file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete index file if it exists.
	indexPath := filepath.Join(store.indexDir, id+".json")
	if err := os.Remove(indexPath); err != nil && !os.IsNotExist(err) {
		http.Error(w, "failed to delete index file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := store.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleGetChunks(w http.ResponseWriter, r *http.Request, store *DocumentStore, id string) {
	doc := store.Get(id)
	if doc == nil {
		http.Error(w, "document not found", http.StatusNotFound)
		return
	}

	indexPath := filepath.Join(store.indexDir, id+".json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "index not found (document may still be indexing)", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to read index: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var idx CombinedIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		http.Error(w, "failed to parse index: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := buildChunkResponse(&idx)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// loadCombinedIndex reads and parses the CombinedIndex JSON for a given document ID.
func loadCombinedIndex(indexDir, docID string) (*CombinedIndex, error) {
	indexPath := filepath.Join(indexDir, docID+".json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("index not found for document %s", docID)
		}
		return nil, fmt.Errorf("failed to read index: %w", err)
	}

	var idx CombinedIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("failed to parse index: %w", err)
	}

	return &idx, nil
}

func handleSearchDoc(w http.ResponseWriter, r *http.Request, store *DocumentStore, id string) {
	doc := store.Get(id)
	if doc == nil {
		http.Error(w, "document not found", http.StatusNotFound)
		return
	}

	if doc.IndexStatus != "ready" {
		http.Error(w, "document index is not ready", http.StatusConflict)
		return
	}

	var body struct {
		Query    string `json:"query"`
		TopK     int    `json:"top_k"`
		Strategy string `json:"strategy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if body.Query == "" {
		http.Error(w, "query is required", http.StatusBadRequest)
		return
	}

	if body.TopK <= 0 {
		body.TopK = 5
	}

	if body.Strategy == "" {
		body.Strategy = "auto"
	}

	idx, err := loadCombinedIndex(store.indexDir, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	embedding, err := GetEmbedding(body.Query)
	if err != nil {
		http.Error(w, "failed to get query embedding: "+err.Error(), http.StatusInternalServerError)
		return
	}

	results := SearchChunks(idx, embedding, body.TopK, body.Strategy)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"results": results,
	})
}

// FindByOriginalName returns the first document matching the given original filename.
func (ds *DocumentStore) FindByOriginalName(name string) *DocumentMeta {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	for _, d := range ds.docs {
		if d.OriginalName == name {
			return d
		}
	}
	return nil
}

// AllReadyDocIDs returns IDs of all documents with index_status "ready".
func (ds *DocumentStore) AllReadyDocIDs() []string {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var ids []string
	for _, d := range ds.docs {
		if d.IndexStatus == "ready" {
			ids = append(ids, d.ID)
		}
	}
	return ids
}

// autoIndexProjectDocs scans docs/*.md files and indexes any that aren't already in the store.
// This enables RAG over project documentation without manual upload.
func autoIndexProjectDocs(store *DocumentStore) {
	matches, err := filepath.Glob("docs/*.md")
	if err != nil || len(matches) == 0 {
		return
	}

	for _, filePath := range matches {
		originalName := filepath.Base(filePath)

		// Skip if already indexed.
		if existing := store.FindByOriginalName(originalName); existing != nil {
			continue
		}

		id, err := generateDocID()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[auto-index] id generation failed for %s: %v\n", originalName, err)
			continue
		}

		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		// Copy file to uploads dir.
		storedFilename := id + "_" + originalName
		destPath := filepath.Join(store.uploadDir, storedFilename)

		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[auto-index] read failed %s: %v\n", filePath, err)
			continue
		}
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "[auto-index] write failed %s: %v\n", destPath, err)
			continue
		}

		doc := &DocumentMeta{
			ID:             id,
			Filename:       storedFilename,
			OriginalName:   originalName,
			ContentType:    "text/markdown",
			Size:           info.Size(),
			UploadedAt:     time.Now(),
			ChunkStrategy:  "all",
			IndexStatus:    "pending",
			EmbeddingModel: "nomic-embed-text",
			ChunkSizeParam: 1000,
			OverlapParam:   200,
		}
		store.Add(doc)

		fmt.Printf("[auto-index] indexing %s (id: %s)\n", originalName, id)
		go runIndexPipeline(store, doc, destPath, 1000, 200)
	}
}

// handleDocsSubroute returns an http.HandlerFunc that dispatches /api/docs/{id}[/action] routes.
func handleDocsSubroute(store *DocumentStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/docs/")
		if path == "" {
			http.Error(w, "document ID required", http.StatusBadRequest)
			return
		}

		// Split into id and optional action.
		parts := strings.SplitN(path, "/", 2)
		id := parts[0]
		action := ""
		if len(parts) == 2 {
			action = parts[1]
		}

		switch action {
		case "chunks":
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			handleGetChunks(w, r, store, id)

		case "search":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			handleSearchDoc(w, r, store, id)

		case "":
			switch r.Method {
			case http.MethodGet:
				handleGetDoc(w, r, store, id)
			case http.MethodDelete:
				handleDeleteDoc(w, r, store, id)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}

		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}
}
