# Data Schemas

Go structs and types used across the backend. All in package `main`.

## Chat & API

### message (api.go)
```go
type message struct {
    Role       string        `json:"role"`        // "user", "assistant", "system"
    Content    any           `json:"content"`     // string or []contentBlock
    Event      *messageEvent `json:"event,omitempty"`
    ApiRequest string        `json:"api_request,omitempty"`
}
```

### contentBlock (agent.go)
```go
type contentBlock struct {
    Type      string          `json:"type"`         // "text", "tool_use", "tool_result"
    Text      string          `json:"text,omitempty"`
    ID        string          `json:"id,omitempty"`
    Name      string          `json:"name,omitempty"`
    Input     json.RawMessage `json:"input,omitempty"`
    ToolUseID string          `json:"tool_use_id,omitempty"`
    Content   string          `json:"content,omitempty"`
    IsError   bool            `json:"is_error,omitempty"`
}
```

### chatRequest (server.go)
```go
type chatRequest struct {
    Message         string   `json:"message"`
    Session         string   `json:"session"`
    Model           string   `json:"model"`
    System          string   `json:"system"`
    MaxTokens       int      `json:"maxTokens"`
    Temperature     float64  `json:"temperature"`
    Strategy        string   `json:"strategy"`
    WindowSize      int      `json:"windowSize"`
    Profile         string   `json:"profile"`
    Project         string   `json:"project"`
    Operator        string   `json:"operator"`
    TaskMode        bool     `json:"taskMode"`
    EnabledTools    []string `json:"enabledTools"`
    Invariants      []string `json:"invariants"`
    McpTools        []string `json:"mcpTools"`
    RagDocIDs       []string `json:"ragDocIds"`
    RagStrategy     string   `json:"ragStrategy"`
    RagTopK         int      `json:"ragTopK"`
    RagThreshold    float64  `json:"ragThreshold"`
    RagQueryRewrite bool     `json:"ragQueryRewrite"`
}
```

### providerSettings (server.go)
```go
type providerSettings struct {
    Provider   string `json:"provider"`   // "claude" | "local" | "railway"
    LocalURL   string `json:"localURL"`
    LocalModel string `json:"localModel"`
    LocalKey   string `json:"localKey"`
}
```

---

## Agent System

### Agent (agent.go)
```go
type Agent struct {
    apiKey      string
    model       string
    maxTurns    int
    maxTokens   int
    system      string
    temperature float64
    tools       []toolDef
    history     []message
    Stats       tokenStats
    workDir     string
    mcpMgr      *MCPManager
    PhaseResult *PhaseResult
    StepResults []StepResult
}
```

### AgentEvent (agent.go)
```go
type AgentEvent struct {
    Type    string          `json:"type"`
    Turn    int             `json:"turn,omitempty"`
    MaxTurn int             `json:"max_turn,omitempty"`
    Tool    string          `json:"tool,omitempty"`
    Input   json.RawMessage `json:"input,omitempty"`
    Output  string          `json:"output,omitempty"`
    IsError bool            `json:"is_error,omitempty"`
    Text    string          `json:"text,omitempty"`
    Usage   *tokenUsage     `json:"usage,omitempty"`
    Stats   *tokenStats     `json:"stats,omitempty"`
}
```

### toolDef (agent.go)
```go
type toolDef struct {
    Name         string         `json:"name"`
    Description  string         `json:"description"`
    InputSchema  any            `json:"input_schema"`
    CacheControl map[string]any `json:"cache_control,omitempty"`
}
```

Built-in tools: `run_shell`, `read_file`. Phase tools: `submit_plan`, `submit_phases`, `submit_validation`, `report_step`.

---

## Task State

### TaskState (taskstate.go)
```go
type TaskState struct {
    Goal              string            `json:"goal"`
    Phase             string            `json:"phase"`              // proposing|planning|executing|validating|done
    Paused            bool              `json:"paused"`
    Steps             []TaskStep        `json:"steps"`
    StepResults       []StepResult      `json:"step_results"`
    Artifacts         map[string]string `json:"artifacts"`
    Feedback          string            `json:"feedback,omitempty"`
    ValidationCount   int               `json:"validation_count"`
    Error             string            `json:"error,omitempty"`
    Invariants        []string          `json:"invariants,omitempty"`
    SandboxDir        string            `json:"sandbox_dir,omitempty"`
    Phases            []PhaseSpec       `json:"phases,omitempty"`
    CurrentPhaseIndex int               `json:"current_phase_index"`
}
```

### PhaseSpec (taskstate.go)
```go
type PhaseSpec struct {
    Name        string `json:"name"`
    Type        string `json:"type"`        // planning|executing|validating
    Description string `json:"description"`
    Status      string `json:"status"`      // pending|active|completed|failed
}
```

### TaskStep, StepResult (taskstate.go)
```go
type TaskStep struct {
    Index       int    `json:"index"`
    Description string `json:"description"`
    Status      string `json:"status"`  // pending|completed|failed
}

type StepResult struct {
    Index  int    `json:"index"`
    Status string `json:"status"`
    Output string `json:"output"`
}
```

---

## Session Persistence

### sessionSettings (history.go)
```go
type sessionSettings struct {
    Model       string  `json:"model,omitempty"`
    Temperature float64 `json:"temperature,omitempty"`
    MaxTokens   int     `json:"max_tokens,omitempty"`
    System      string  `json:"system,omitempty"`
    Strategy    string  `json:"strategy,omitempty"`
    WindowSize  int     `json:"window_size,omitempty"`
    Profile     string  `json:"profile,omitempty"`
    Project     string  `json:"project,omitempty"`
    Operator    string  `json:"operator,omitempty"`
}
```

### sessionFile (history.go)
```go
type sessionFile struct {
    SavedAt      time.Time
    Messages     []message
    Summary      string
    Settings     *sessionSettings
    Stats        *sessionStats
    Facts        map[string]string
    Branches     []branch
    ActiveBranch string
    TaskState    *TaskState
}
```

Stored as JSON in `.chat_history/<name>.json`.

---

## Documents & RAG

### DocumentMeta (docs.go)
```go
type DocumentMeta struct {
    ID             string    `json:"id"`
    Filename       string    `json:"filename"`
    OriginalName   string    `json:"original_name"`
    ContentType    string    `json:"content_type"`
    Size           int64     `json:"size"`
    UploadedAt     time.Time `json:"uploaded_at"`
    ChunkCount     int       `json:"chunk_count"`
    ChunkStrategy  string    `json:"chunk_strategy"`
    IndexStatus    string    `json:"index_status"`    // pending|indexing|ready|error
    IndexError     string    `json:"index_error,omitempty"`
    EmbeddingModel string    `json:"embedding_model"`
    ChunkSizeParam int       `json:"chunk_size_param"`
    OverlapParam   int       `json:"overlap_param"`
}
```

### Chunk (chunker.go)
```go
type Chunk struct {
    ID       string            `json:"id"`
    DocID    string            `json:"doc_id"`
    Text     string            `json:"text"`
    Index    int               `json:"index"`
    Metadata map[string]string `json:"metadata"`
}
```

### CombinedIndex (indexer.go)
```go
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
```

### SearchResult (similarity.go)
```go
type SearchResult struct {
    Chunk    Chunk   `json:"chunk"`
    Score    float64 `json:"score"`
    Strategy string  `json:"strategy"`
}
```

---

## MCP

### MCPServerConfig (mcp.go)
```go
type MCPServerConfig struct {
    Command string            `json:"command"`
    Args    []string          `json:"args"`
    Env     map[string]string `json:"env,omitempty"`
}
```

### MCPToolInfo (mcp.go)
```go
type MCPToolInfo struct {
    Server      string `json:"server"`
    Name        string `json:"name"`
    Description string `json:"description"`
    InputSchema any    `json:"inputSchema"`
}
```

Tool naming convention: `server__toolname` (double underscore separator).

---

## Tokens & Pricing

### tokenUsage (tokens.go)
```go
type tokenUsage struct {
    InputTokens        int `json:"input"`
    OutputTokens       int `json:"output"`
    CacheCreationInput int `json:"cache_creation_input,omitempty"`
    CacheReadInput     int `json:"cache_read_input,omitempty"`
}
```

### ModelPricing (models.go)
```go
const (
    ModelSonnet  = "claude-sonnet-4-6"
    ModelHaiku   = "claude-haiku-4-5"
    ModelOpus    = "claude-opus-4-6"
    DefaultModel = ModelSonnet
)

type ModelPricing struct {
    CostIn       float64  // USD per 1M input tokens
    CostOut      float64  // USD per 1M output tokens
    CacheWriteIn float64
    CacheReadIn  float64
}
```
