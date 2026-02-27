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

export async function deleteSessionAPI(name: string) {
  const resp = await fetch(`/api/sessions/${encodeURIComponent(name)}`, {
    method: 'DELETE',
  })
  if (!resp.ok) {
    throw new Error(`Failed to delete session: ${resp.statusText}`)
  }
}
