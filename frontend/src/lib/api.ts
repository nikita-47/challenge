export async function fetchSessions(): Promise<string[]> {
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

export async function fetchBranchesAPI(session: string) {
  const resp = await fetch(`/api/sessions/${encodeURIComponent(session)}/branches`)
  if (!resp.ok) {
    throw new Error(`Failed to fetch branches: ${resp.statusText}`)
  }
  return resp.json()
}
