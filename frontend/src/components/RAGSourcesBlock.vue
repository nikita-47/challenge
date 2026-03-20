<script setup lang="ts">
import { ref } from 'vue'
import type { RAGSource } from '@/lib/types'
import { Collapsible, CollapsibleTrigger, CollapsibleContent } from '@/components/ui/collapsible'
import { Badge } from '@/components/ui/badge'

const props = defineProps<{
  sources: RAGSource[]
}>()

const isOpen = ref(true)

function scoreColor(score: number): string {
  if (score >= 0.7) {
    return 'text-emerald-400'
  }
  if (score >= 0.4) {
    return 'text-amber-400'
  }
  return 'text-zinc-500'
}
</script>

<template>
  <Collapsible v-model:open="isOpen" class="mt-2">
    <div class="border border-border/50 bg-background/50 text-xs rounded-sm">
      <CollapsibleTrigger class="w-full flex items-center gap-2 px-2 py-1.5 text-left hover:bg-muted/30 transition-colors">
        <span class="text-muted-foreground font-mono text-[10px]">sources</span>
        <span class="text-muted-foreground/80">({{ props.sources.length }})</span>
        <span class="text-muted-foreground/50 text-[10px] ml-auto">{{ isOpen ? '[-]' : '[+]' }}</span>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div class="border-t border-border/50 divide-y divide-border/30">
          <div
            v-for="source in props.sources"
            :key="source.ref"
            class="px-2 py-1.5 flex items-start gap-2"
          >
            <Badge
              variant="default"
              class="shrink-0 h-4 min-w-4 px-1 font-mono text-[10px] leading-none flex items-center justify-center"
            >
              {{ source.ref }}
            </Badge>
            <div class="flex flex-col gap-0.5 min-w-0 flex-1">
              <span class="text-muted-foreground/90 font-medium truncate" :title="source.source">
                {{ source.source }}
              </span>
              <span class="text-muted-foreground/50 font-mono text-[10px]">
                {{ source.chunk }}
              </span>
            </div>
            <span
              class="shrink-0 font-mono text-[10px] font-semibold"
              :class="scoreColor(source.score)"
            >
              {{ Math.round(source.score * 100) }}%
            </span>
          </div>
        </div>
      </CollapsibleContent>
    </div>
  </Collapsible>
</template>
