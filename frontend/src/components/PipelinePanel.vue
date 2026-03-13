<script setup lang="ts">
import { ref, computed } from 'vue'
import { usePipelineStore } from '@/stores/pipeline'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Input } from '@/components/ui/input'
import type { PipelineStepStatus } from '@/lib/types'

const pipeline = usePipelineStore()

const query = ref('')

const FLOW_STEPS = ['search', 'summarize', 'save'] as const

const activeSteps = computed(() => {
  if (!pipeline.activeRun) {
    return FLOW_STEPS.map((name) => ({ name, status: 'pending' as PipelineStepStatus, output: undefined, error: undefined, started_at: undefined, finished_at: undefined }))
  }
  return FLOW_STEPS.map((name) => {
    const found = pipeline.activeRun!.steps.find((s) => s.name === name)
    return found ?? { name, status: 'pending' as PipelineStepStatus, output: undefined, error: undefined, started_at: undefined, finished_at: undefined }
  })
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

async function handleRun() {
  const q = query.value.trim()
  if (!q || pipeline.loading) {
    return
  }
  await pipeline.startPipeline(q)
}

function selectRun(id: string) {
  pipeline.loadStatus(id)
}
</script>

<template>
  <div class="flex flex-col h-full">
    <!-- Query input -->
    <div class="p-3 border-b border-primary/20 space-y-2">
      <div class="text-[10px] text-primary/60 uppercase tracking-wider font-mono">
        // pipeline
      </div>
      <div class="flex gap-2">
        <Input
          v-model="query"
          placeholder="search query..."
          class="flex-1 h-7 text-xs bg-background border-primary/30 font-mono"
          @keydown.enter="handleRun"
        />
        <Button
          size="sm"
          class="h-7 px-3 text-xs shrink-0"
          :disabled="pipeline.loading || !query.trim()"
          @click="handleRun"
        >
          {{ pipeline.loading ? '...' : 'run' }}
        </Button>
      </div>
      <div v-if="pipeline.error" class="text-xs text-red-400 border border-red-400/30 px-2 py-1 bg-red-400/5 font-mono">
        {{ pipeline.error }}
      </div>
    </div>

    <ScrollArea class="flex-1">
      <div class="p-3 space-y-4">
        <!-- Active run flow diagram -->
        <div class="space-y-1">
          <div class="text-[10px] text-muted-foreground font-mono mb-2">
            <span v-if="pipeline.activeRun">
              run:
              <span class="text-primary font-mono">{{ pipeline.activeRun.id.slice(0, 8) }}</span>
              <span class="mx-1">—</span>
              <span :class="runStatusClass(pipeline.activeRun.status)">{{ pipeline.activeRun.status }}</span>
              <span v-if="pipeline.polling" class="ml-2 text-primary/50 animate-pulse">polling...</span>
            </span>
            <span v-else class="text-muted-foreground/50">// no active run</span>
          </div>

          <!-- Flow steps -->
          <div class="flex flex-col items-center gap-0">
            <template v-for="(step, idx) in activeSteps" :key="step.name">
              <!-- Step box -->
              <details class="w-full group">
                <summary
                  class="flex items-center justify-between px-3 py-2 border text-xs font-mono cursor-pointer select-none list-none transition-colors"
                  :class="stepClass(step.status)"
                >
                  <span class="font-semibold">{{ step.name }}</span>
                  <div class="flex items-center gap-2 text-[10px] shrink-0">
                    <span v-if="step.started_at">{{ formatTime(step.started_at) }}</span>
                    <span
                      class="px-1 border"
                      :class="stepClass(step.status)"
                    >{{ step.status }}</span>
                  </div>
                </summary>
                <!-- Step output / error -->
                <div
                  v-if="step.output || step.error"
                  class="border border-t-0 border-primary/10 bg-background px-3 py-2"
                >
                  <pre
                    v-if="step.output"
                    class="text-[10px] text-muted-foreground font-mono whitespace-pre-wrap break-words leading-relaxed"
                  >{{ step.output }}</pre>
                  <pre
                    v-if="step.error"
                    class="text-[10px] text-red-400 font-mono whitespace-pre-wrap break-words leading-relaxed"
                  >{{ step.error }}</pre>
                </div>
              </details>

              <!-- Arrow between steps -->
              <div
                v-if="idx < activeSteps.length - 1"
                class="text-muted-foreground/40 font-mono text-xs leading-none py-0.5"
              >
                &darr;
              </div>
            </template>
          </div>

          <!-- Output file -->
          <div v-if="pipeline.activeRun?.output_file" class="mt-2 flex items-center gap-2 text-[10px] font-mono border border-green-400/20 bg-green-400/5 px-2 py-1.5">
            <span class="text-green-400/70">output:</span>
            <span class="text-green-400 truncate">{{ pipeline.activeRun.output_file }}</span>
          </div>
        </div>

        <!-- Divider -->
        <div class="border-t border-primary/10" />

        <!-- Previous runs list -->
        <div class="space-y-1">
          <div class="text-[10px] text-primary/60 uppercase tracking-wider font-mono flex items-center justify-between">
            <span>// runs</span>
            <button
              class="text-primary/50 hover:text-primary transition-colors"
              @click="pipeline.loadList()"
            >
              ↺
            </button>
          </div>
          <div v-if="pipeline.runs.length === 0" class="text-xs text-muted-foreground/50 font-mono py-2">
            // no runs yet
          </div>
          <div
            v-for="run in pipeline.runs"
            :key="run.id"
            class="flex items-start justify-between gap-2 px-2 py-1.5 border border-transparent hover:border-primary/20 hover:bg-primary/5 cursor-pointer transition-colors"
            :class="pipeline.activeRun?.id === run.id ? 'border-primary/30 bg-primary/5' : ''"
            @click="selectRun(run.id)"
          >
            <div class="min-w-0 flex-1 space-y-0.5">
              <div class="text-xs font-mono text-foreground truncate">{{ run.query }}</div>
              <div class="flex items-center gap-2 text-[10px] font-mono">
                <span class="text-muted-foreground/60">{{ run.id.slice(0, 8) }}</span>
                <span class="text-muted-foreground/40">{{ formatDate(run.created_at) }}</span>
              </div>
            </div>
            <span
              class="text-[10px] font-mono px-1 border shrink-0 mt-0.5"
              :class="stepClass(run.status)"
            >{{ run.status }}</span>
          </div>
        </div>
      </div>
    </ScrollArea>
  </div>
</template>
