export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  toolCalls?: ToolCall[]
  isStreaming?: boolean
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
  input: number
  output: number
  Usage?: TokenUsage
  Stats?: TokenStats
}

export interface DoneEvent {
  type: 'done'
  Turn?: number
  Stats?: TokenStats
}

export interface ErrorEvent {
  type: 'error'
  message?: string
  Text?: string
}

export interface TurnEvent {
  type: 'turn'
  Turn: number
  MaxTurn: number
}

export interface ThinkingEvent {
  type: 'thinking'
  Text: string
}

export interface ToolCallEvent {
  type: 'tool_call'
  Tool: string
  Input: Record<string, unknown>
}

export interface ToolResultEvent {
  type: 'tool_result'
  Tool: string
  Output: string
  IsError: boolean
}

export interface CompressEvent {
  type: 'compress'
  messageCount: number
  summaryLen: number
  tokensSaved: number
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
  | { type: 'text'; Text: string }
