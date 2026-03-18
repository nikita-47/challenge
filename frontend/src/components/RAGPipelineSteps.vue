<script setup lang="ts">
import { ref, computed } from 'vue'
import type { RAGStep, RAGStepName } from '@/lib/types'
import { Collapsible, CollapsibleTrigger, CollapsibleContent } from '@/components/ui/collapsible'

const props = defineProps<{
  steps: RAGStep[]
  rewrittenQuery?: string
}>()

const isOpen = ref(false)

const stepLabels: Record<RAGStepName, string> = {
  rewrite: 'Rewrite query',
  embed: 'Embed query',
  search: 'Search chunks',
  filter: 'Filter by relevance',
  inject: 'Inject context',
}

const statusColors: Record<string, string> = {
  done: 'text-emerald-400',
  running: 'text-amber-400',
  skipped: 'text-zinc-500',
  error: 'text-red-400',
}

function statusIcon(status: string): string {
  if (status === 'done') {
    return '✓'
  }
  if (status === 'running') {
    return '↻'
  }
  if (status === 'skipped') {
    return '→'
  }
  return '✗'
}

function isSpinning(status: string): boolean {
  return status === 'running'
}

function stepDetail(step: RAGStep): string | null {
  if (!step.detail) {
    return null
  }

  if (step.step === 'filter') {
    const d = step.detail as { passed?: number; rejected?: number; threshold?: number }
    const passed = d.passed ?? 0
    const rejected = d.rejected ?? 0
    const threshold = d.threshold ?? 0
    return `${passed} passed, ${rejected} rejected (threshold: ${threshold.toFixed(2)})`
  }

  if (step.step === 'search') {
    const d = step.detail as { total?: number }
    const total = d.total ?? 0
    return `${total} chunks found`
  }

  return null
}

const doneCount = computed(() => props.steps.filter((s) => s.status === 'done').length)
</script>

<template>
  <Collapsible v-model:open="isOpen" class="mb-2">
    <div class="border border-border/50 bg-background/50 text-xs rounded-sm">
      <CollapsibleTrigger class="w-full flex items-center gap-2 px-2 py-1.5 text-left hover:bg-muted/30 transition-colors">
        <span class="text-muted-foreground font-mono text-[10px]">rag pipeline</span>
        <span class="text-muted-foreground/80">
          {{ doneCount }}/{{ steps.length }} steps
        </span>
        <span class="text-muted-foreground/50 text-[10px] ml-auto">{{ isOpen ? '[-]' : '[+]' }}</span>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div class="border-t border-border/50 px-2 py-1 flex flex-col gap-0.5">
          <div
            v-for="step in steps"
            :key="step.step"
            class="flex items-start gap-2 py-1"
          >
            <span
              class="shrink-0 w-3 text-center font-mono"
              :class="[statusColors[step.status] ?? 'text-zinc-500', isSpinning(step.status) ? 'animate-spin inline-block' : '']"
            >
              {{ statusIcon(step.status) }}
            </span>
            <div class="flex flex-col gap-0.5 min-w-0">
              <span :class="statusColors[step.status] ?? 'text-zinc-500'">
                {{ stepLabels[step.step] ?? step.step }}
              </span>
              <!-- rewrite step: show original → rewritten -->
              <template v-if="step.step === 'rewrite' && step.status === 'done' && rewrittenQuery">
                <span class="text-muted-foreground/60 break-words">→ {{ rewrittenQuery }}</span>
              </template>
              <!-- generic detail for filter/search steps -->
              <template v-else-if="stepDetail(step)">
                <span class="text-muted-foreground/60">{{ stepDetail(step) }}</span>
              </template>
            </div>
          </div>
        </div>
      </CollapsibleContent>
    </div>
  </Collapsible>
</template>
