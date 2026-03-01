import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { ChatMessage, ToolCall, TokenUsage, ChatSettings } from '@/lib/types'
import { streamRequest } from '@/composables/useSSE'

export const useChatStore = defineStore('chat', () => {
  const messages = ref<ChatMessage[]>([])
  const isStreaming = ref(false)
  const currentSession = ref('default')
  const usage = ref<TokenUsage>({ input: 0, output: 0 })
  const totalUsage = ref<TokenUsage>({ input: 0, output: 0 })
  const exchanges = ref(0)
  const error = ref<string | null>(null)
  const settings = ref<ChatSettings | null>(null)

  let abortController: AbortController | null = null

  const totalCost = computed(() => {
    return totalUsage.value.input * 3.0 / 1e6 + totalUsage.value.output * 15.0 / 1e6
  })

  function setSettings(s: ChatSettings | null) {
    settings.value = s
  }

  function clearMessages() {
    messages.value = []
    usage.value = { input: 0, output: 0 }
    totalUsage.value = { input: 0, output: 0 }
    exchanges.value = 0
    error.value = null
    settings.value = null
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

    const url = '/api/chat'
    const body = {
      message: text,
      session: currentSession.value,
      ...(settings.value && {
        system: settings.value.system,
        maxTokens: settings.value.maxTokens,
        temperature: settings.value.temperature,
      }),
    }

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
            msg.content += event.text + '\n'
            break

          case 'tool_call': {
            const tc: ToolCall = {
              tool: event.tool,
              input: event.input as Record<string, unknown>,
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
              (tc) => tc.tool === event.tool && tc.output === undefined,
            )
            if (last) {
              last.output = event.output
              last.isError = event.is_error
            }
            break
          }

          case 'usage':
            if (event.usage) {
              const u = event.usage
              usage.value = { input: u.input, output: u.output }
              totalUsage.value.input += u.input
              totalUsage.value.output += u.output
              exchanges.value++
            }
            break

          case 'done':
            msg.isStreaming = false
            break

          case 'error':
            error.value = event.message ?? event.text ?? 'Unknown error'
            msg.isStreaming = false
            break

          case 'compress': {
            const compressMsg: ChatMessage = {
              role: 'system',
              content: '',
              event: {
                type: 'compress',
                messageCount: event.messageCount,
                tokensSaved: event.tokensSaved,
              },
            }
            const insertIndex = messages.value.length - 2
            if (insertIndex >= 0) {
              messages.value.splice(insertIndex, 0, compressMsg)
            } else {
              messages.value.unshift(compressMsg)
            }
            break
          }
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
    currentSession,
    usage,
    totalUsage,
    exchanges,
    totalCost,
    error,
    settings,
    sendMessage,
    clearMessages,
    setMessages,
    setSettings,
    stopStreaming,
  }
})
