import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { ChatMessage, ToolCall, TokenUsage } from '@/lib/types'
import { streamRequest } from '@/composables/useSSE'

export const useChatStore = defineStore('chat', () => {
  const messages = ref<ChatMessage[]>([])
  const isStreaming = ref(false)
  const isAgentMode = ref(false)
  const currentSession = ref('default')
  const usage = ref<TokenUsage>({ input: 0, output: 0 })
  const totalUsage = ref<TokenUsage>({ input: 0, output: 0 })
  const exchanges = ref(0)
  const error = ref<string | null>(null)

  let abortController: AbortController | null = null

  const totalCost = computed(() => {
    return totalUsage.value.input * 3.0 / 1e6 + totalUsage.value.output * 15.0 / 1e6
  })

  function clearMessages() {
    messages.value = []
    usage.value = { input: 0, output: 0 }
    totalUsage.value = { input: 0, output: 0 }
    exchanges.value = 0
    error.value = null
  }

  function setMessages(msgs: ChatMessage[]) {
    messages.value = msgs
  }

  function stopStreaming() {
    if (abortController) {
      abortController.abort()
      abortController = null
    }
    isStreaming.value = false
  }

  async function sendMessage(text: string) {
    error.value = null
    messages.value.push({ role: 'user', content: text })

    const assistantMsg: ChatMessage = {
      role: 'assistant',
      content: '',
      isStreaming: true,
    }
    messages.value.push(assistantMsg)

    isStreaming.value = true
    abortController = new AbortController()

    const url = isAgentMode.value ? '/api/agent' : '/api/chat'
    const body = isAgentMode.value
      ? { task: text, session: currentSession.value }
      : { message: text, session: currentSession.value }

    let pendingToolCalls: ToolCall[] = []

    try {
      await streamRequest(url, body, (event) => {
        const msg = messages.value[messages.value.length - 1]
        if (!msg) {
          return
        }

        switch (event.type) {
          case 'text_delta':
            msg.content += event.text
            break

          case 'text':
            // Agent goal echo, ignore
            break

          case 'turn':
            // Agent turn marker — could show in UI
            break

          case 'thinking':
            msg.content += event.Text + '\n'
            break

          case 'tool_call': {
            const tc: ToolCall = {
              tool: event.Tool,
              input: event.Input as Record<string, unknown>,
            }
            pendingToolCalls.push(tc)
            if (!msg.toolCalls) {
              msg.toolCalls = []
            }
            msg.toolCalls.push(tc)
            break
          }

          case 'tool_result': {
            const last = pendingToolCalls.find(
              (tc) => tc.tool === event.Tool && tc.output === undefined,
            )
            if (last) {
              last.output = event.Output
              last.isError = event.IsError
            }
            break
          }

          case 'usage':
            if (event.input !== undefined) {
              usage.value = { input: event.input, output: event.output }
              totalUsage.value.input += event.input
              totalUsage.value.output += event.output
              exchanges.value++
            } else if (event.Usage) {
              usage.value = { input: event.Usage.input, output: event.Usage.output }
            }
            break

          case 'done':
            msg.isStreaming = false
            if (event.Stats) {
              totalUsage.value = {
                input: event.Stats.TotalInput,
                output: event.Stats.TotalOutput,
              }
              exchanges.value = event.Stats.Exchanges
            }
            break

          case 'error':
            error.value = event.message ?? event.Text ?? 'Unknown error'
            msg.isStreaming = false
            break

          case 'compress':
            // Could show compression info
            break
        }
      }, abortController.signal)
    } catch (e) {
      if (e instanceof DOMException && e.name === 'AbortError') {
        // User cancelled
      } else {
        error.value = e instanceof Error ? e.message : 'Unknown error'
      }
    } finally {
      isStreaming.value = false
      const msg = messages.value[messages.value.length - 1]
      if (msg) {
        msg.isStreaming = false
      }
      pendingToolCalls = []
    }
  }

  return {
    messages,
    isStreaming,
    isAgentMode,
    currentSession,
    usage,
    totalUsage,
    exchanges,
    totalCost,
    error,
    sendMessage,
    clearMessages,
    setMessages,
    stopStreaming,
  }
})
