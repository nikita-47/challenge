import type { MCPServerStatus, MCPToolInfo, DocumentMeta, ChunkIndex } from '@/lib/types'

export interface SessionInfo {
  name: string
  profile?: string
  project?: string
  operator?: string
}

export async function fetchSessions(): Promise<SessionInfo[]> {
  const resp = await fetch('/api/sessions')
  if (!resp.ok) {
    throw new Error(`Failed to fetch sessions: ${resp.statusText}`)
  }
  const data = await resp.json()
  return data ?? []
}

export async function fetchSession(name: string) {
  const resp = await fetch(`/api/sessions/${encodeURIComponent(name)}`)
  if (!resp.ok) {
    throw new Error(`Failed to load session: ${resp.statusText}`)
  }
  return resp.json()
}

export async function renameSessionAPI(oldName: string, newName: string) {
  const resp = await fetch(`/api/sessions/${encodeURIComponent(oldName)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ newName }),
  })
  if (!resp.ok) {
    throw new Error(`Failed to rename session: ${resp.statusText}`)
  }
  return resp.json()
}

export interface AppConfig {
  model: string
  maxTokens: number
  temperature: number
  system: string
}

export async function fetchConfig(): Promise<AppConfig> {
  const resp = await fetch('/api/config')
  if (!resp.ok) {
    throw new Error(`Failed to fetch config: ${resp.statusText}`)
  }
  return resp.json()
}

export async function deleteSessionAPI(name: string) {
  const resp = await fetch(`/api/sessions/${encodeURIComponent(name)}`, {
    method: 'DELETE',
  })
  if (!resp.ok) {
    throw new Error(`Failed to delete session: ${resp.statusText}`)
  }
}

export async function createBranchAPI(session: string, branchName: string) {
  const resp = await fetch(`/api/sessions/${encodeURIComponent(session)}/branches`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: branchName }),
  })
  if (!resp.ok) {
    throw new Error(`Failed to create branch: ${resp.statusText}`)
  }
  return resp.json()
}

export async function switchBranchAPI(session: string, branchName: string) {
  const resp = await fetch(`/api/sessions/${encodeURIComponent(session)}/branch`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: branchName }),
  })
  if (!resp.ok) {
    throw new Error(`Failed to switch branch: ${resp.statusText}`)
  }
  return resp.json()
}

export async function fetchSessionRaw(name: string): Promise<string> {
  const resp = await fetch(`/api/sessions/${encodeURIComponent(name)}/raw`)
  if (!resp.ok) {
    throw new Error(`Failed to fetch raw session: ${resp.statusText}`)
  }
  return resp.text()
}

// ─── Memory API ──────────────────────────────────────────────────────────────

export async function fetchProfiles(): Promise<string[]> {
  const resp = await fetch('/api/memory/profiles')
  if (!resp.ok) {
    throw new Error(`Failed to fetch profiles: ${resp.statusText}`)
  }
  const data = await resp.json()
  return data ?? []
}

export async function fetchProfile(name: string): Promise<{ name: string; content: string }> {
  const resp = await fetch(`/api/memory/profiles/${encodeURIComponent(name)}`)
  if (!resp.ok) {
    throw new Error(`Failed to fetch profile: ${resp.statusText}`)
  }
  return resp.json()
}

export async function createProfile(name: string, content: string) {
  const resp = await fetch('/api/memory/profiles', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, content }),
  })
  if (!resp.ok) {
    throw new Error(`Failed to create profile: ${resp.statusText}`)
  }
  return resp.json()
}

export async function updateProfile(name: string, content: string) {
  const resp = await fetch(`/api/memory/profiles/${encodeURIComponent(name)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content }),
  })
  if (!resp.ok) {
    throw new Error(`Failed to update profile: ${resp.statusText}`)
  }
  return resp.json()
}

export async function deleteProfileAPI(name: string) {
  const resp = await fetch(`/api/memory/profiles/${encodeURIComponent(name)}`, {
    method: 'DELETE',
  })
  if (!resp.ok) {
    throw new Error(`Failed to delete profile: ${resp.statusText}`)
  }
}

export async function fetchProjects(): Promise<string[]> {
  const resp = await fetch('/api/memory/projects')
  if (!resp.ok) {
    throw new Error(`Failed to fetch projects: ${resp.statusText}`)
  }
  const data = await resp.json()
  return data ?? []
}

export async function fetchProject(name: string): Promise<{ name: string; content: string }> {
  const resp = await fetch(`/api/memory/projects/${encodeURIComponent(name)}`)
  if (!resp.ok) {
    throw new Error(`Failed to fetch project: ${resp.statusText}`)
  }
  return resp.json()
}

export async function createProject(name: string, content: string) {
  const resp = await fetch('/api/memory/projects', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, content }),
  })
  if (!resp.ok) {
    throw new Error(`Failed to create project: ${resp.statusText}`)
  }
  return resp.json()
}

export async function updateProject(name: string, content: string) {
  const resp = await fetch(`/api/memory/projects/${encodeURIComponent(name)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content }),
  })
  if (!resp.ok) {
    throw new Error(`Failed to update project: ${resp.statusText}`)
  }
  return resp.json()
}

export async function deleteProjectAPI(name: string) {
  const resp = await fetch(`/api/memory/projects/${encodeURIComponent(name)}`, {
    method: 'DELETE',
  })
  if (!resp.ok) {
    throw new Error(`Failed to delete project: ${resp.statusText}`)
  }
}

// ─── Operator Memory API ─────────────────────────────────────────────────────

export async function fetchOperators(): Promise<string[]> {
  const resp = await fetch('/api/memory/operators')
  if (!resp.ok) {
    throw new Error(`Failed to fetch operators: ${resp.statusText}`)
  }
  const data = await resp.json()
  return data ?? []
}

export async function fetchOperator(name: string): Promise<{ name: string; content: string }> {
  const resp = await fetch(`/api/memory/operators/${encodeURIComponent(name)}`)
  if (!resp.ok) {
    throw new Error(`Failed to fetch operator: ${resp.statusText}`)
  }
  return resp.json()
}

export async function createOperator(name: string, content: string) {
  const resp = await fetch('/api/memory/operators', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, content }),
  })
  if (!resp.ok) {
    throw new Error(`Failed to create operator: ${resp.statusText}`)
  }
  return resp.json()
}

export async function updateOperator(name: string, content: string) {
  const resp = await fetch(`/api/memory/operators/${encodeURIComponent(name)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content }),
  })
  if (!resp.ok) {
    throw new Error(`Failed to update operator: ${resp.statusText}`)
  }
  return resp.json()
}

export async function deleteOperatorAPI(name: string) {
  const resp = await fetch(`/api/memory/operators/${encodeURIComponent(name)}`, {
    method: 'DELETE',
  })
  if (!resp.ok) {
    throw new Error(`Failed to delete operator: ${resp.statusText}`)
  }
}

export async function fetchBranchesAPI(session: string) {
  const resp = await fetch(`/api/sessions/${encodeURIComponent(session)}/branches`)
  if (!resp.ok) {
    throw new Error(`Failed to fetch branches: ${resp.statusText}`)
  }
  return resp.json()
}

// ─── Provider Settings API ────────────────────────────────────────────────────

export interface ProviderSettings {
  provider: 'claude' | 'local' | 'railway'
  localURL: string
  localModel: string
  localKey: string
}

export async function fetchSettings(): Promise<ProviderSettings> {
  const resp = await fetch('/api/settings')
  if (!resp.ok) {
    throw new Error(`Failed to fetch settings: ${resp.statusText}`)
  }
  return resp.json()
}

export async function updateSettings(settings: ProviderSettings): Promise<ProviderSettings> {
  const resp = await fetch('/api/settings', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(settings),
  })
  if (!resp.ok) {
    throw new Error(`Failed to update settings: ${resp.statusText}`)
  }
  return resp.json()
}

// ─── MCP API ─────────────────────────────────────────────────────────────────

export async function fetchMCPServers(): Promise<MCPServerStatus[]> {
  const resp = await fetch('/api/mcp/servers')
  if (!resp.ok) {
    throw new Error(`Failed to fetch MCP servers: ${resp.statusText}`)
  }
  const data = await resp.json()
  return data ?? []
}

export async function fetchMCPTools(server?: string): Promise<MCPToolInfo[]> {
  const url = server ? `/api/mcp/tools?server=${encodeURIComponent(server)}` : '/api/mcp/tools'
  const resp = await fetch(url)
  if (!resp.ok) {
    throw new Error(`Failed to fetch MCP tools: ${resp.statusText}`)
  }
  const data = await resp.json()
  return data ?? []
}

export async function connectMCPServer(name: string) {
  const resp = await fetch(`/api/mcp/servers/${encodeURIComponent(name)}/connect`, { method: 'POST' })
  if (!resp.ok) {
    throw new Error(`Failed to connect: ${resp.statusText}`)
  }
  return resp.json()
}

export async function disconnectMCPServer(name: string) {
  const resp = await fetch(`/api/mcp/servers/${encodeURIComponent(name)}/disconnect`, { method: 'POST' })
  if (!resp.ok) {
    throw new Error(`Failed to disconnect: ${resp.statusText}`)
  }
  return resp.json()
}

export async function reloadMCPConfig() {
  const resp = await fetch('/api/mcp/reload', { method: 'POST' })
  if (!resp.ok) {
    throw new Error(`Failed to reload: ${resp.statusText}`)
  }
  return resp.json()
}

// ─── Documents API ───────────────────────────────────────────────────────

export async function fetchDocs(): Promise<DocumentMeta[]> {
  const resp = await fetch('/api/docs')
  if (!resp.ok) {
    throw new Error(`Failed to fetch documents: ${resp.statusText}`)
  }
  const data = await resp.json()
  return data ?? []
}

export async function uploadDoc(file: File, chunkSize?: number, overlap?: number): Promise<DocumentMeta> {
  const form = new FormData()
  form.append('file', file)
  if (chunkSize !== undefined) {
    form.append('chunk_size', String(chunkSize))
  }
  if (overlap !== undefined) {
    form.append('overlap', String(overlap))
  }
  const resp = await fetch('/api/docs/upload', { method: 'POST', body: form })
  if (!resp.ok) {
    throw new Error(`Failed to upload document: ${resp.statusText}`)
  }
  return resp.json()
}

export async function fetchDoc(id: string): Promise<DocumentMeta> {
  const resp = await fetch(`/api/docs/${encodeURIComponent(id)}`)
  if (!resp.ok) {
    throw new Error(`Failed to fetch document: ${resp.statusText}`)
  }
  return resp.json()
}

export async function deleteDoc(id: string): Promise<void> {
  const resp = await fetch(`/api/docs/${encodeURIComponent(id)}`, { method: 'DELETE' })
  if (!resp.ok) {
    throw new Error(`Failed to delete document: ${resp.statusText}`)
  }
}

export async function fetchDocChunks(id: string): Promise<ChunkIndex> {
  const resp = await fetch(`/api/docs/${encodeURIComponent(id)}/chunks`)
  if (!resp.ok) {
    throw new Error(`Failed to fetch chunk index: ${resp.statusText}`)
  }
  return resp.json()
}


export async function callMCPTool(server: string, tool: string, args: Record<string, unknown>): Promise<string> {
  const resp = await fetch('/api/mcp/tools/call', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ server, tool, arguments: args }),
  })
  if (!resp.ok) {
    throw new Error(`MCP tool call failed: ${resp.statusText}`)
  }
  const data = await resp.json()
  // MCP CallToolResult format: { content: [{type: "text", text: "..."}], isError: bool }
  if (Array.isArray(data.content)) {
    const texts = data.content
      .filter((c: Record<string, unknown>) => c.type === 'text')
      .map((c: Record<string, unknown>) => c.text)
    return texts.join('\n')
  }
  return data.result ?? JSON.stringify(data)
}
