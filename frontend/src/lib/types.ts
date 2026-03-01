export type ContextStrategy = 'summary' | 'window' | 'facts' | 'branch'

export interface ChatSettings {
  model: string
  temperature: number
  maxTokens: number
  system: string
  strategy?: ContextStrategy
  windowSize?: number
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
  | { type: 'text'; Text: string }
