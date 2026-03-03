import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { ChatMessage, ToolCall, TokenUsage, ChatSettings, BranchInfo } from '@/lib/types'
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
  const hasSummary = ref(false)
  const compressionCount = ref(0)

  const facts = ref<Record<string, string>>({})
  const branches = ref<BranchInfo[]>([])
  const activeBranch = ref('main')

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
    hasSummary.value = false
    compressionCount.value = 0
    tokensSaved.value = 0
    facts.value = {}
    branches.value = []
    activeBranch.value = 'main'
  }

  const tokensSaved = ref(0)

  function setStats(stats: { total_input: number; total_output: number; exchanges: number; tokens_saved: number }) {
    totalUsage.value = { input: stats.total_input, output: stats.total_output }
    exchanges.value = stats.exchanges
    tokensSaved.value = stats.tokens_saved
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
        model: settings.value.model,
        system: settings.value.system,
        maxTokens: settings.value.maxTokens,
        temperature: settings.value.temperature,
        strategy: settings.value.strategy,
        windowSize: settings.value.windowSize,
        profile: settings.value.profile,
        project: settings.value.project,
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

          case 'api_request': {
            for (let i = messages.value.length - 1; i >= 0; i--) {
              const m = messages.value[i]
              if (m && m.role === 'user') {
                m.apiRequest = event.text
                break
              }
            }
            break
          }

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

          case 'facts_updated':
            facts.value = event.facts
            break

          case 'compress': {
            hasSummary.value = true
            compressionCount.value++
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
    setStats,
    stopStreaming,
    hasSummary,
    compressionCount,
    tokensSaved,
    facts,
    branches,
    activeBranch,
  }
})
