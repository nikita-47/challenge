import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { PipelineRun } from '@/lib/types'
import { callMCPTool } from '@/lib/api'

function parsePipelineRun(raw: unknown): PipelineRun | null {
  if (typeof raw !== 'object' || raw === null) {
    return null
  }
  const obj = raw as Record<string, unknown>
  return {
    id: String(obj.id ?? ''),
    query: String(obj.query ?? ''),
    source: String(obj.source ?? ''),
    status: (obj.status as PipelineRun['status']) ?? 'pending',
    steps: Array.isArray(obj.steps) ? obj.steps.map((s: unknown) => {
      const step = s as Record<string, unknown>
      return {
        name: String(step.name ?? ''),
        status: (step.status as PipelineRun['status']) ?? 'pending',
        started_at: step.started_at ? String(step.started_at) : undefined,
        finished_at: step.finished_at ? String(step.finished_at) : undefined,
        output: step.output ? String(step.output) : undefined,
        error: step.error ? String(step.error) : undefined,
      }
    }) : [],
    output_file: obj.output_file ? String(obj.output_file) : undefined,
    created_at: String(obj.created_at ?? new Date().toISOString()),
  }
}

function tryParseJSON(text: string): unknown {
  try {
    return JSON.parse(text)
  } catch {
    return null
  }
}

export const usePipelineStore = defineStore('pipeline', () => {
  const runs = ref<PipelineRun[]>([])
  const activeRun = ref<PipelineRun | null>(null)
  const loading = ref(false)
  const polling = ref(false)
  const error = ref<string | null>(null)
  let pollTimer: ReturnType<typeof setInterval> | null = null

  async function loadList() {
    loading.value = true
    error.value = null
    try {
      const result = await callMCPTool('pipeline', 'pipe_list', {})
      const parsed = tryParseJSON(result)
      if (Array.isArray(parsed)) {
        runs.value = parsed.map(parsePipelineRun).filter((r): r is PipelineRun => r !== null)
      } else if (parsed && typeof parsed === 'object') {
        const obj = parsed as Record<string, unknown>
        const list = obj.runs ?? obj.pipelines ?? obj.data
        if (Array.isArray(list)) {
          runs.value = list.map(parsePipelineRun).filter((r): r is PipelineRun => r !== null)
        } else {
          runs.value = []
        }
      } else {
        runs.value = []
      }
    } catch (e) {
      console.error('Failed to load pipeline list:', e)
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      loading.value = false
    }
  }

  async function loadStatus(id?: string) {
    const targetId = id ?? activeRun.value?.id
    if (!targetId) {
      return
    }
    try {
      const result = await callMCPTool('pipeline', 'pipe_status', { id: targetId })
      const parsed = tryParseJSON(result)
      const run = parsePipelineRun(parsed)
      if (run) {
        activeRun.value = run
        const idx = runs.value.findIndex((r) => r.id === run.id)
        if (idx !== -1) {
          runs.value[idx] = run
        }
      }
    } catch (e) {
      console.error('Failed to load pipeline status:', e)
      error.value = e instanceof Error ? e.message : String(e)
    }
  }

  async function startPipeline(query: string, count = 5) {
    loading.value = true
    error.value = null
    try {
      const result = await callMCPTool('pipeline', 'pipe_run', { query, count })

      // pipe_run returns: "Pipeline started: <id>\nQuery: ...\n..."
      // Try JSON first, then parse text
      let id: string | null = null
      const parsed = tryParseJSON(result)
      if (parsed && typeof parsed === 'object') {
        const obj = parsed as Record<string, unknown>
        id = String(obj.id ?? obj.run_id ?? '')
      } else if (typeof result === 'string') {
        const match = result.match(/Pipeline started:\s*(\S+)/)
        if (match?.[1]) {
          id = match[1]
        }
      }

      if (id) {
        startPolling(id)
        await loadList()
      }
    } catch (e) {
      console.error('Failed to start pipeline:', e)
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      loading.value = false
    }
  }

  function startPolling(id: string) {
    stopPolling()
    polling.value = true
    pollTimer = setInterval(async () => {
      await loadStatus(id)
      const run = activeRun.value
      if (run && (run.status === 'done' || run.status === 'error')) {
        stopPolling()
        await loadList()
      }
    }, 2000)
  }

  function stopPolling() {
    if (pollTimer !== null) {
      clearInterval(pollTimer)
      pollTimer = null
    }
    polling.value = false
  }

  async function deleteRun(id: string) {
    try {
      await callMCPTool('pipeline', 'pipe_delete', { id })
      if (activeRun.value?.id === id) {
        activeRun.value = null
        stopPolling()
      }
      runs.value = runs.value.filter((r) => r.id !== id)
    } catch (e) {
      console.error('Failed to delete pipeline run:', e)
      error.value = e instanceof Error ? e.message : String(e)
    }
  }

  return { runs, activeRun, loading, polling, error, loadList, loadStatus, startPipeline, startPolling, stopPolling, deleteRun }
})
