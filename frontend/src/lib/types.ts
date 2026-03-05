export type ContextStrategy = 'summary' | 'window' | 'facts' | 'branch'

// ─── Task State Machine ─────────────────────────────────────────────────────

export type TaskPhase = 'planning' | 'executing' | 'validating' | 'done' | 'paused'
export type TaskStepStatus = 'pending' | 'in_progress' | 'completed' | 'failed'

export interface TaskStep {
  index: number
  description: string
  status: TaskStepStatus
  result?: string
}

export interface TaskState {
  goal: string
  phase: TaskPhase
  steps: TaskStep[]
  current_step: number
  expected_action?: string
  paused_at_phase?: string
  error?: string
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

export interface ChatMessage {
  role: 'user' | 'assistant' | 'system'
  content: string
  toolCalls?: ToolCall[]
  isStreaming?: boolean
  event?: SystemEvent
  apiRequest?: string
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
  | { type: 'memory_updated' }
  | { type: 'text'; Text: string }
