import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { MCPServerStatus, MCPToolInfo } from '@/lib/types'
import {
  fetchMCPServers,
  fetchMCPTools,
  connectMCPServer,
  disconnectMCPServer,
  reloadMCPConfig,
} from '@/lib/api'

export const useMCPStore = defineStore('mcp', () => {
  const servers = ref<MCPServerStatus[]>([])
  const tools = ref<MCPToolInfo[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function loadServers() {
    try {
      servers.value = await fetchMCPServers()
    } catch (e) {
      console.error('Failed to load MCP servers:', e)
      error.value = e instanceof Error ? e.message : String(e)
    }
  }

  async function loadTools(server?: string) {
    try {
      tools.value = await fetchMCPTools(server)
    } catch (e) {
      console.error('Failed to load MCP tools:', e)
      error.value = e instanceof Error ? e.message : String(e)
    }
  }

  async function connect(name: string) {
    loading.value = true
    error.value = null
    try {
      await connectMCPServer(name)
      await Promise.all([loadServers(), loadTools()])
    } catch (e) {
      console.error('Failed to connect MCP server:', e)
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      loading.value = false
    }
  }

  async function disconnect(name: string) {
    loading.value = true
    error.value = null
    try {
      await disconnectMCPServer(name)
      await Promise.all([loadServers(), loadTools()])
    } catch (e) {
      console.error('Failed to disconnect MCP server:', e)
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      loading.value = false
    }
  }

  async function reload() {
    loading.value = true
    error.value = null
    try {
      await reloadMCPConfig()
      await Promise.all([loadServers(), loadTools()])
    } catch (e) {
      console.error('Failed to reload MCP config:', e)
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      loading.value = false
    }
  }

  return { servers, tools, loading, error, loadServers, loadTools, connect, disconnect, reload }
})
