import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchSessions, fetchSession, deleteSessionAPI, renameSessionAPI } from '@/lib/api'
import type { SessionInfo } from '@/lib/api'
import { useChatStore } from './chat'
import type { ChatMessage, ChatSettings } from '@/lib/types'

export const useSessionsStore = defineStore('sessions', () => {
  const sessions = ref<SessionInfo[]>([])
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
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const msgs: ChatMessage[] = (data.messages ?? []).map((m: any) => {
        const msg: ChatMessage = {
          role: m.role as ChatMessage['role'],
          content: typeof m.content === 'string' ? m.content : JSON.stringify(m.content),
        }
        if (m.event) {
          msg.event = {
            type: m.event.type,
            messageCount: m.event.message_count ?? m.event.messageCount,
            summaryLen: m.event.summary_len ?? m.event.summaryLen,
            tokensSaved: m.event.tokens_saved ?? m.event.tokensSaved,
          }
        }
        if (m.api_request) {
          msg.apiRequest = m.api_request
        }
        return msg
      })
      chat.setMessages(msgs)
      chat.currentSession = name
      if (data.settings) {
        chat.setSettings({
          model: data.settings.model ?? '',
          temperature: data.settings.temperature ?? 0.7,
          maxTokens: data.settings.max_tokens ?? data.settings.maxTokens ?? 1024,
          system: data.settings.system ?? '',
          strategy: data.settings.strategy ?? undefined,
          windowSize: data.settings.window_size ?? data.settings.windowSize ?? undefined,
          profile: data.settings.profile ?? undefined,
          project: data.settings.project ?? undefined,
        })
      } else {
        chat.setSettings(null)
      }
      if (data.stats) {
        chat.setStats(data.stats)
      }
      chat.hasSummary = !!data.summary
      chat.compressionCount = msgs.filter((m: ChatMessage) => m.event?.type === 'compress').length
      chat.facts = data.facts ?? {}
      chat.branches = (data.branches ?? []).map((b: any) => ({
        name: b.name,
        forkIndex: b.forkIndex ?? b.fork_index ?? 0,
        messageCount: b.messageCount ?? b.message_count ?? 0,
        createdAt: b.createdAt ?? b.created_at ?? '',
      }))
      chat.activeBranch = data.activeBranch ?? data.active_branch ?? 'main'
      chat.taskState = data.taskState ?? null
    } catch (e) {
      console.error('Failed to load session:', e)
    }
  }

  async function deleteSession(name: string) {
    try {
      await deleteSessionAPI(name)
      sessions.value = sessions.value.filter((s) => s.name !== name)
      const chat = useChatStore()
      if (chat.currentSession === name) {
        chat.clearMessages()
        chat.currentSession = 'default'
      }
    } catch (e) {
      console.error('Failed to delete session:', e)
    }
  }

  async function renameSession(oldName: string, newName: string) {
    await renameSessionAPI(oldName, newName)
    const entry = sessions.value.find((s) => s.name === oldName)
    if (entry) {
      entry.name = newName
    }
    const chat = useChatStore()
    if (chat.currentSession === oldName) {
      chat.currentSession = newName
    }
  }

  function newChat(name?: string, chatSettings?: ChatSettings) {
    const chat = useChatStore()
    const sessionName = name?.trim() || `chat-${Date.now()}`
    chat.clearMessages()
    chat.currentSession = sessionName
    if (chatSettings) {
      chat.setSettings(chatSettings)
    }
    sessions.value.unshift({
      name: sessionName,
      profile: chatSettings?.profile,
      project: chatSettings?.project,
    })
  }

  return {
    sessions,
    loading,
    loadList,
    loadSession,
    deleteSession,
    renameSession,
    newChat,
  }
})
