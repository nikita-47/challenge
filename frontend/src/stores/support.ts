import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { SupportMessage } from '@/lib/types'
import { streamRequest } from '@/composables/useSSE'

export const useSupportStore = defineStore('support', () => {
  const isOpen = ref(false)
  const messages = ref<SupportMessage[]>([])
  const sending = ref(false)
  const currentTicketId = ref<string | null>(null)
  const error = ref<string | null>(null)
  let abortController: AbortController | null = null

  function toggle() {
    isOpen.value = !isOpen.value
  }

  async function sendMessage(text: string) {
    if (!text.trim() || sending.value) {
      return
    }

    error.value = null
    messages.value.push({ role: 'user', content: text })
    messages.value.push({ role: 'assistant', content: '', isStreaming: true })
    sending.value = true
    abortController = new AbortController()

    // Send last 10 messages as history (excluding current streaming assistant message)
    const history = messages.value
      .filter(m => !m.isStreaming)
      .slice(-10)
      .map(m => ({ role: m.role, content: m.content }))

    // Remove the user message we just added from history (it's sent as "message")
    history.pop()

    try {
      await streamRequest(
        '/api/support/chat',
        {
          message: text,
          ticketId: currentTicketId.value ?? undefined,
          history,
        },
        (event) => {
          const lastMsg = messages.value[messages.value.length - 1]
          switch (event.type) {
            case 'text_delta': {
              const e = event as { type: 'text_delta'; text: string }
              if (lastMsg && lastMsg.role === 'assistant') {
                lastMsg.content += e.text
              }
              break
            }
            case 'error': {
              error.value = (event as { type: 'error'; message?: string }).message ?? 'Support error'
              break
            }
            case 'done': {
              if (lastMsg) {
                lastMsg.isStreaming = false
              }
              break
            }
          }
        },
        abortController.signal,
      )
    } catch (e) {
      if ((e as Error).name !== 'AbortError') {
        error.value = e instanceof Error ? e.message : String(e)
      }
      // Remove empty assistant message on error
      const lastMsg = messages.value[messages.value.length - 1]
      if (lastMsg?.role === 'assistant' && !lastMsg.content) {
        messages.value.pop()
      }
    } finally {
      sending.value = false
      abortController = null
      // Ensure streaming flag is cleared
      const lastMsg = messages.value[messages.value.length - 1]
      if (lastMsg) {
        lastMsg.isStreaming = false
      }
    }
  }

  function cancelMessage() {
    abortController?.abort()
  }

  function clearChat() {
    messages.value = []
    currentTicketId.value = null
    error.value = null
  }

  return {
    isOpen,
    messages,
    sending,
    currentTicketId,
    error,
    toggle,
    sendMessage,
    cancelMessage,
    clearChat,
  }
})
