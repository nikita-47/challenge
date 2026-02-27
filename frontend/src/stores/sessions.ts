import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchSessions, fetchSession, deleteSessionAPI } from '@/lib/api'
import { useChatStore } from './chat'
import type { ChatMessage } from '@/lib/types'

export const useSessionsStore = defineStore('sessions', () => {
  const sessions = ref<string[]>([])
  const loading = ref(false)

  async function loadList() {
    loading.value = true
    try {
      sessions.value = await fetchSessions()
    } catch {
      sessions.value = []
    } finally {
      loading.value = false
    }
  }

  async function loadSession(name: string) {
    const chat = useChatStore()
    try {
      const data = await fetchSession(name)
      const msgs: ChatMessage[] = (data.messages ?? []).map((m: { role: string; content: unknown }) => ({
        role: m.role as 'user' | 'assistant',
        content: typeof m.content === 'string' ? m.content : JSON.stringify(m.content),
      }))
      chat.setMessages(msgs)
      chat.currentSession = name
    } catch (e) {
      console.error('Failed to load session:', e)
    }
  }

  async function deleteSession(name: string) {
    try {
      await deleteSessionAPI(name)
      sessions.value = sessions.value.filter((s) => s !== name)
      const chat = useChatStore()
      if (chat.currentSession === name) {
        chat.clearMessages()
        chat.currentSession = 'default'
      }
    } catch (e) {
      console.error('Failed to delete session:', e)
    }
  }

  function newChat() {
    const chat = useChatStore()
    const name = `chat-${Date.now()}`
    chat.clearMessages()
    chat.currentSession = name
  }

  return {
    sessions,
    loading,
    loadList,
    loadSession,
    deleteSession,
    newChat,
  }
})
