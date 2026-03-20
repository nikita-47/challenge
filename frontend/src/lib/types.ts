export type ContextStrategy = 'summary' | 'window' | 'facts' | 'branch'

// ─── Task State Machine ─────────────────────────────────────────────────────

export type TaskPhase = 'proposing' | 'planning' | 'executing' | 'validating' | 'done'
export type TaskStepStatus = 'pending' | 'in_progress' | 'completed' | 'failed'

export interface PhaseSpec {
  name: string
  type: 'planning' | 'executing' | 'validating'
  description: string
  status: 'pending' | 'active' | 'completed' | 'failed'
}

export interface TaskStep {
  index: number
  description: string
  status: TaskStepStatus
  result?: string
}

export interface StepResult {
  index: number
  status: 'completed' | 'failed'
  output: string
}

export interface TaskState {
  goal: string
  phase: TaskPhase
  paused: boolean
  invariants?: string[]
  steps: TaskStep[]
  step_results: StepResult[]
  artifacts: Record<string, string>
  feedback?: string
  validation_count: number
  error?: string
  phases?: PhaseSpec[]
  current_phase_index?: number
}

export interface ChatSettings {
  model: string
  temperature: number
  maxTokens: number
  system: string
  strategy?: ContextStrategy
  windowSize?: number
  profile?: string
  project?: string
  operator?: string
}

export interface MemoryFile {
  name: string
  content: string
}

export interface BranchInfo {
  name: string
  forkIndex: number
  messageCount: number
  createdAt: string
}

export interface SystemEvent {
  type: string
  messageCount?: number
  summaryLen?: number
  tokensSaved?: number
}

export interface RAGSource {
  ref: number
  source: string
  chunk: string
  score: number
}

export interface RAGSearchResult {
  chunk: {
    id: string
    doc_id: string
    text: string
    index: number
    metadata: Record<string, string>
  }
  score: number
  strategy: string
  doc_name: string
}

// RAG Pipeline step types
export type RAGStepName = 'rewrite' | 'embed' | 'search' | 'filter' | 'inject'
export type RAGStepStatus = 'running' | 'done' | 'skipped' | 'error'

export interface RAGStep {
  step: RAGStepName
  status: RAGStepStatus
  detail?: unknown
}

export interface RAGStepEvent {
  type: 'rag_step'
  step: RAGStepName
  status: RAGStepStatus
  detail?: unknown
}

export interface ChatMessage {
  role: 'user' | 'assistant' | 'system'
  content: string
  toolCalls?: ToolCall[]
  isStreaming?: boolean
  event?: SystemEvent
  apiRequest?: string
  ragContext?: RAGSearchResult[]
  ragSteps?: RAGStep[]
  ragAllResults?: RAGSearchResult[]
  ragRejected?: RAGSearchResult[]
  ragRewrittenQuery?: string
  ragThreshold?: number
  ragNoContext?: boolean
  ragSources?: RAGSource[]
}

export interface ToolCall {
  tool: string
  input: Record<string, unknown>
  output?: string
  isError?: boolean
}

export interface TokenUsage {
  input: number
  output: number
}

export interface TokenStats {
  TotalInput: number
  TotalOutput: number
  Exchanges: number
  TokensSaved: number
}

// SSE event types from Go backend
export interface TextDeltaEvent {
  type: 'text_delta'
  text: string
}

export interface UsageEvent {
  type: 'usage'
  input?: number
  output?: number
  usage?: TokenUsage
  stats?: TokenStats
}

export interface DoneEvent {
  type: 'done'
  turn?: number
  stats?: TokenStats
}

export interface ErrorEvent {
  type: 'error'
  message?: string
  text?: string
}

export interface TurnEvent {
  type: 'turn'
  turn: number
  max_turn: number
}

export interface ThinkingEvent {
  type: 'thinking'
  text: string
}

export interface ToolCallEvent {
  type: 'tool_call'
  tool: string
  input: Record<string, unknown>
}

export interface ToolResultEvent {
  type: 'tool_result'
  tool: string
  output: string
  is_error: boolean
}

export interface CompressEvent {
  type: 'compress'
  messageCount: number
  summaryLen: number
  tokensSaved: number
}

export interface FactsUpdatedEvent {
  type: 'facts_updated'
  facts: Record<string, string>
}

export interface ApiRequestEvent {
  type: 'api_request'
  text: string
}

export interface TaskStateEvent {
  type: 'task_state'
  text: string
}

export interface StepResultEvent {
  type: 'step_result'
  text: string
}

export interface RAGContextEvent {
  type: 'rag_context'
  results: RAGSearchResult[]
  all_results?: RAGSearchResult[]
  rejected?: RAGSearchResult[]
  rewritten_query?: string
  threshold?: number
  no_context?: boolean
}

export type SSEEvent =
  | TextDeltaEvent
  | UsageEvent
  | DoneEvent
  | ErrorEvent
  | TurnEvent
  | ThinkingEvent
  | ToolCallEvent
  | ToolResultEvent
  | CompressEvent
  | FactsUpdatedEvent
  | ApiRequestEvent
  | TaskStateEvent
  | StepResultEvent
  | RAGContextEvent
  | RAGStepEvent
  | { type: 'memory_updated' }
  | { type: 'text'; Text: string }

// ─── Pipeline Types ─────────────────────────────────────────────────────────
export type PipelineStepStatus = 'pending' | 'running' | 'done' | 'error'

export interface PipelineStep {
  name: string
  status: PipelineStepStatus
  started_at?: string
  finished_at?: string
  output?: string
  error?: string
}

export interface PipelineRun {
  id: string
  query: string
  source: string
  status: PipelineStepStatus
  steps: PipelineStep[]
  output_file?: string
  created_at: string
}

// ─── Document Indexing Types ─────────────────────────────────────────────
export type IndexStatus = 'pending' | 'indexing' | 'ready' | 'error'
export type ChunkStrategy = 'structure' | 'sentence' | 'size'

export interface DocumentMeta {
  id: string
  filename: string
  original_name: string
  content_type: string
  size: number
  uploaded_at: string
  chunk_count: number
  chunk_strategy: ChunkStrategy
  index_status: IndexStatus
  index_error?: string
  embedding_model: string
  chunk_size_param: number
  overlap_param: number
}

export interface DocumentChunk {
  id: string
  doc_id: string
  text: string
  index: number
  metadata: Record<string, string>
  similarity_to_next: number | null
  embedding_dim: number
}

export interface StrategyResult {
  chunks: DocumentChunk[]
  avg_similarity: number
  min_similarity: number
  max_similarity: number
}

export interface ChunkIndex {
  doc_id: string
  filename: string
  embedding_model: string
  created_at: string
  size: StrategyResult
  sentence: StrategyResult
  structure: StrategyResult
  semantic: StrategyResult
}

// ─── MCP Types ──────────────────────────────────────────────────────────────
export interface MCPServerStatus {
  name: string
  connected: boolean
  toolsCount: number
  error?: string
}

export interface MCPToolInfo {
  server: string
  name: string
  description: string
  inputSchema: unknown
}
