<script setup lang="ts">
import { ref, computed } from 'vue'
import type { ToolCall } from '@/lib/types'
import { Collapsible, CollapsibleTrigger, CollapsibleContent } from '@/components/ui/collapsible'

const props = defineProps<{
  toolCall: ToolCall
}>()

const expanded = ref(false)

const inputDisplay = computed(() => {
  if (props.toolCall.tool === 'run_shell') {
    return `$ ${(props.toolCall.input as { command?: string }).command ?? ''}`
  }
  if (props.toolCall.tool === 'read_file') {
    return (props.toolCall.input as { path?: string }).path ?? ''
  }
  return JSON.stringify(props.toolCall.input)
})
</script>

<template>
  <Collapsible v-model:open="expanded">
    <div
      class="rounded-md border text-xs"
      :class="toolCall.isError ? 'border-destructive bg-destructive/10' : 'border-border bg-black/20'"
    >
      <CollapsibleTrigger class="w-full flex items-center gap-2 px-2 py-1.5 text-left hover:bg-accent/50 transition-colors rounded-t-md">
        <span class="text-amber-400 font-mono">{{ toolCall.tool }}</span>
        <span class="text-muted-foreground truncate flex-1 font-mono">{{ inputDisplay }}</span>
        <span class="text-muted-foreground">{{ expanded ? '▼' : '▶' }}</span>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div v-if="toolCall.output" class="border-t border-border px-2 py-1.5">
          <pre class="whitespace-pre-wrap text-muted-foreground font-mono max-h-48 overflow-y-auto">{{ toolCall.output }}</pre>
        </div>
      </CollapsibleContent>
      <div v-if="!toolCall.output && !expanded" class="px-2 py-1 text-muted-foreground">
        <span class="animate-pulse">Running...</span>
      </div>
    </div>
  </Collapsible>
</template>
