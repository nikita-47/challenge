<script setup lang="ts">
import { computed } from 'vue'
import type { RAGSearchResult } from '@/lib/types'

const props = defineProps<{
  passed: RAGSearchResult[]
  rejected: RAGSearchResult[]
  threshold: number
}>()

const isSplitView = computed(() => props.threshold > 0 && props.rejected.length > 0)

const allChunks = computed(() => {
  return [...props.passed, ...props.rejected].sort((a, b) => b.score - a.score)
})

const rejectedIds = computed(() => new Set(props.rejected.map((r) => r.chunk.id)))

function scoreColor(score: number): string {
  if (score >= 0.7) {
    return 'text-emerald-400'
  }
  if (score >= 0.4) {
    return 'text-amber-400'
  }
  return 'text-zinc-400'
}

function truncate(text: string, max = 100): string {
  if (text.length <= max) {
    return text
  }
  return text.slice(0, max) + '…'
}
</script>

<template>
  <!-- Split view when threshold filtering was applied -->
  <div v-if="isSplitView" class="grid grid-cols-2 gap-2 text-xs">
    <!-- Left: before filter -->
    <div class="flex flex-col gap-1">
      <p class="text-muted-foreground/70 font-medium py-0.5">
        Before filter ({{ allChunks.length }})
      </p>
      <div
        v-for="(result, i) in allChunks"
        :key="i"
        class="flex flex-col gap-0.5 px-1.5 py-1 border border-border/30 rounded-sm"
        :class="rejectedIds.has(result.chunk.id) ? 'opacity-40' : ''"
      >
        <div class="flex items-center gap-1">
          <span
            class="truncate text-muted-foreground/70 flex-1"
            :title="result.doc_name"
          >
            {{ result.doc_name }}
          </span>
          <span class="shrink-0 font-mono" :class="scoreColor(result.score)">
            {{ (result.score * 100).toFixed(0) }}%
          </span>
        </div>
        <p
          class="text-muted-foreground/60 leading-snug"
          :class="rejectedIds.has(result.chunk.id) ? 'line-through' : ''"
        >
          {{ truncate(result.chunk.text) }}
        </p>
      </div>
    </div>

    <!-- Right: after filter -->
    <div class="flex flex-col gap-1">
      <p class="text-muted-foreground/70 font-medium py-0.5">
        After filter ({{ passed.length }})
      </p>
      <div
        v-for="(result, i) in passed"
        :key="i"
        class="flex flex-col gap-0.5 px-1.5 py-1 border border-border/30 rounded-sm"
      >
        <div class="flex items-center gap-1">
          <span
            class="truncate text-muted-foreground/70 flex-1"
            :title="result.doc_name"
          >
            {{ result.doc_name }}
          </span>
          <span class="shrink-0 font-mono" :class="scoreColor(result.score)">
            {{ (result.score * 100).toFixed(0) }}%
          </span>
        </div>
        <p class="text-muted-foreground/60 leading-snug">
          {{ truncate(result.chunk.text) }}
        </p>
      </div>
      <p v-if="passed.length === 0" class="text-muted-foreground/40 italic py-1">
        all chunks filtered out
      </p>
    </div>
  </div>

  <!-- Single column view: no threshold or no rejected chunks -->
  <div v-else class="flex flex-col divide-y divide-border/30">
    <div
      v-for="(result, i) in passed"
      :key="i"
      class="flex flex-col gap-0.5 px-2 py-1.5"
    >
      <div class="flex items-center gap-2">
        <span
          class="text-muted-foreground/70 font-medium truncate flex-1"
          :title="result.doc_name"
        >
          {{ result.doc_name }}
        </span>
        <span class="font-mono shrink-0" :class="scoreColor(result.score)">
          {{ (result.score * 100).toFixed(0) }}%
        </span>
      </div>
      <p class="text-muted-foreground/60 leading-snug">
        {{ truncate(result.chunk.text) }}
      </p>
    </div>
  </div>
</template>
