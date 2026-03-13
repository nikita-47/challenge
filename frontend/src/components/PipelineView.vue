<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { usePipelineStore } from '@/stores/pipeline'
import { useUIStore } from '@/stores/ui'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import type { PipelineStep, PipelineStepStatus } from '@/lib/types'

const pipeline = usePipelineStore()
const ui = useUIStore()

const query = ref('')
const selectedStepName = ref<string | null>(null)

const FLOW_STEPS = ['search', 'summarize', 'save'] as const

const activeSteps = computed(() => {
  if (!pipeline.activeRun) {
    return FLOW_STEPS.map((name) => ({
      name,
      status: 'pending' as PipelineStepStatus,
      output: undefined as string | undefined,
      error: undefined as string | undefined,
      started_at: undefined as string | undefined,
      finished_at: undefined as string | undefined,
    }))
  }
  return FLOW_STEPS.map((name) => {
    const found = pipeline.activeRun!.steps.find((s) => s.name === name)
    return found ?? {
      name,
      status: 'pending' as PipelineStepStatus,
      output: undefined as string | undefined,
      error: undefined as string | undefined,
      started_at: undefined as string | undefined,
      finished_at: undefined as string | undefined,
    }
  })
})

const selectedStep = computed<(typeof activeSteps.value)[number] | null>(() => {
  if (!selectedStepName.value) {
    return null
  }
  return activeSteps.value.find((s) => s.name === selectedStepName.value) ?? null
})

function stepClass(status: PipelineStepStatus): string {
  if (status === 'running') {
    return 'bg-primary/20 text-primary border-primary/40 animate-pulse'
  }
  if (status === 'done') {
    return 'bg-green-400/10 text-green-400 border-green-400/30'
  }
  if (status === 'error') {
    return 'bg-red-400/10 text-red-400 border-red-400/30'
  }
  return 'text-muted-foreground border-border'
}

function runStatusClass(status: PipelineStepStatus): string {
  if (status === 'running') {
    return 'text-primary'
  }
  if (status === 'done') {
    return 'text-green-400'
  }
  if (status === 'error') {
    return 'text-red-400'
  }
  return 'text-muted-foreground'
}

function formatTime(iso?: string): string {
  if (!iso) {
    return ''
  }
  const d = new Date(iso)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function formatDate(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

function stepDuration(step: PipelineStep | { started_at?: string; finished_at?: string }): string {
  if (!step.started_at || !step.finished_at) {
    return ''
  }
  const ms = new Date(step.finished_at).getTime() - new Date(step.started_at).getTime()
  if (ms < 1000) {
    return `${ms}ms`
  }
  return `${(ms / 1000).toFixed(1)}s`
}

async function handleRun() {
  const q = query.value.trim()
  if (!q || pipeline.loading) {
    return
  }
  selectedStepName.value = null
  await pipeline.startPipeline(q)
}

function selectRun(id: string) {
  selectedStepName.value = null
  pipeline.loadStatus(id)
}

function selectStep(name: string) {
  if (selectedStepName.value === name) {
    selectedStepName.value = null
  } else {
    selectedStepName.value = name
  }
}

onMounted(() => {
  pipeline.loadList()
})
</script>

<template>
  <div class="flex flex-col h-screen w-screen bg-background font-mono overflow-hidden">
    <!-- Header bar -->
    <div class="flex items-center gap-4 px-4 py-2 border-b border-primary/20 bg-card shrink-0">
      <div class="text-sm text-primary font-semibold tracking-widest shrink-0">
        // pipeline
      </div>

      <div class="flex-1 flex items-center gap-2 justify-center max-w-xl mx-auto">
        <Input
          v-model="query"
          placeholder="search query..."
          class="flex-1 h-7 text-xs bg-background border-primary/30 font-mono"
          @keydown.enter="handleRun"
        />
        <Button
          size="sm"
          class="h-7 px-4 text-xs shrink-0"
          :disabled="pipeline.loading || !query.trim()"
          @click="handleRun"
        >
          {{ pipeline.loading ? '...' : 'run' }}
        </Button>
      </div>

      <Button
        variant="ghost"
        size="sm"
        class="h-7 text-xs text-muted-foreground hover:text-foreground shrink-0"
        @click="ui.setView('chat')"
      >
        ← back to chat
      </Button>
    </div>

    <!-- Error bar -->
    <div
      v-if="pipeline.error"
      class="shrink-0 text-xs text-red-400 border-b border-red-400/30 px-4 py-1.5 bg-red-400/5 font-mono"
    >
      {{ pipeline.error }}
    </div>

    <!-- Body -->
    <div class="flex flex-1 min-h-0">
      <!-- Left column: runs list -->
      <div class="w-72 shrink-0 border-r border-primary/20 flex flex-col">
        <div class="flex items-center justify-between px-3 py-2 border-b border-primary/10">
          <span class="text-[10px] text-primary/60 uppercase tracking-wider">// runs</span>
          <button
            class="text-primary/50 hover:text-primary transition-colors text-sm"
            title="Refresh"
            @click="pipeline.loadList()"
          >
            ↺
          </button>
        </div>
        <ScrollArea class="flex-1">
          <div class="p-2 space-y-1">
            <div
              v-if="pipeline.runs.length === 0"
              class="text-xs text-muted-foreground/50 font-mono py-3 px-1"
            >
              // no runs yet
            </div>
            <div
              v-for="run in pipeline.runs"
              :key="run.id"
              class="group flex items-start justify-between gap-2 px-2 py-2 border cursor-pointer transition-colors"
              :class="pipeline.activeRun?.id === run.id
                ? 'border-primary/40 bg-primary/5 text-primary'
                : 'border-transparent hover:border-primary/20 hover:bg-primary/5'"
              @click="selectRun(run.id)"
            >
              <div class="min-w-0 flex-1 space-y-1">
                <div class="text-xs font-mono text-foreground truncate">{{ run.query }}</div>
                <div class="flex items-center gap-2 text-[10px] font-mono text-muted-foreground/60">
                  <span>{{ run.id.slice(0, 8) }}</span>
                  <span class="text-muted-foreground/40">{{ formatDate(run.created_at) }}</span>
                </div>
              </div>
              <span
                class="text-[10px] font-mono px-1 border shrink-0 mt-0.5"
                :class="stepClass(run.status)"
              >{{ run.status }}</span>
              <button
                v-if="run.status !== 'running'"
                class="text-[10px] text-muted-foreground/30 hover:text-red-400 transition-colors opacity-0 group-hover:opacity-100 ml-1 shrink-0"
                title="delete run"
                @click.stop="pipeline.deleteRun(run.id)"
              >
                ×
              </button>
            </div>
          </div>
        </ScrollArea>
      </div>

      <!-- Center area -->
      <div class="flex-1 flex flex-col min-w-0">
        <!-- Active run meta -->
        <div class="shrink-0 px-4 py-2 border-b border-primary/10 flex items-center gap-3 text-[11px] font-mono text-muted-foreground">
          <template v-if="pipeline.activeRun">
            <span>run: <span class="text-primary">{{ pipeline.activeRun.id.slice(0, 8) }}</span></span>
            <span class="text-muted-foreground/40">—</span>
            <span :class="runStatusClass(pipeline.activeRun.status)">{{ pipeline.activeRun.status }}</span>
            <span v-if="pipeline.polling" class="text-primary/50 animate-pulse">polling...</span>
            <span class="text-muted-foreground/40 truncate max-w-xs">{{ pipeline.activeRun.query }}</span>
          </template>
          <template v-else>
            <span class="text-muted-foreground/40">// no active run — select from list or run a query</span>
          </template>
        </div>

        <!-- Flow diagram: horizontal steps -->
        <div class="shrink-0 px-4 py-4 border-b border-primary/10">
          <div class="flex items-center gap-0">
            <template v-for="(step, idx) in activeSteps" :key="step.name">
              <!-- Step box -->
              <button
                class="flex flex-col items-start px-4 py-2.5 border text-xs font-mono transition-colors min-w-[120px] cursor-pointer"
                :class="[
                  stepClass(step.status),
                  selectedStepName === step.name ? 'ring-1 ring-primary/60' : '',
                ]"
                @click="selectStep(step.name)"
              >
                <span class="font-semibold text-sm">{{ step.name }}</span>
                <div class="flex items-center gap-2 mt-1 text-[10px] opacity-80">
                  <span
                    class="px-1 border"
                    :class="stepClass(step.status)"
                  >{{ step.status }}</span>
                  <span v-if="stepDuration(step)" class="text-muted-foreground/60">{{ stepDuration(step) }}</span>
                </div>
                <div v-if="step.started_at" class="text-[10px] opacity-60 mt-0.5">
                  {{ formatTime(step.started_at) }}
                </div>
              </button>

              <!-- Arrow between steps -->
              <div
                v-if="idx < activeSteps.length - 1"
                class="text-muted-foreground/40 font-mono text-base px-2 shrink-0 select-none"
              >
                →
              </div>
            </template>
          </div>

          <!-- Output file badge -->
          <div
            v-if="pipeline.activeRun?.output_file"
            class="mt-3 inline-flex items-center gap-2 text-[10px] font-mono border border-green-400/20 bg-green-400/5 px-2 py-1"
          >
            <span class="text-green-400/70">output:</span>
            <span class="text-green-400">{{ pipeline.activeRun.output_file }}</span>
          </div>
        </div>

        <!-- Output panel -->
        <div class="flex-1 min-h-0 flex flex-col">
          <div class="px-4 py-2 border-b border-primary/10 text-[10px] text-primary/50 uppercase tracking-wider shrink-0">
            <template v-if="selectedStep">
              // {{ selectedStep.name }} output
            </template>
            <template v-else>
              // select a step to view output
            </template>
          </div>

          <ScrollArea class="flex-1">
            <div class="p-4">
              <template v-if="selectedStep">
                <div v-if="!selectedStep.output && !selectedStep.error" class="text-xs text-muted-foreground/50 font-mono">
                  // no output yet
                </div>
                <pre
                  v-if="selectedStep.output"
                  class="text-xs text-muted-foreground font-mono whitespace-pre-wrap break-words leading-relaxed"
                >{{ selectedStep.output }}</pre>
                <pre
                  v-if="selectedStep.error"
                  class="text-xs text-red-400 font-mono whitespace-pre-wrap break-words leading-relaxed mt-2"
                >{{ selectedStep.error }}</pre>
              </template>
              <div v-else class="text-xs text-muted-foreground/30 font-mono">
                // click a step in the flow diagram above to inspect its output
              </div>
            </div>
          </ScrollArea>
        </div>
      </div>
    </div>
  </div>
</template>
