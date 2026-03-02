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
      class="border text-xs"
      :class="toolCall.isError ? 'border-destructive/50 bg-destructive/5' : 'border-border bg-background'"
    >
      <CollapsibleTrigger class="w-full flex items-center gap-2 px-2 py-1.5 text-left hover:bg-primary/5 transition-colors">
        <span class="text-primary font-mono">{{ toolCall.tool }}</span>
        <span class="text-muted-foreground truncate flex-1 font-mono">{{ inputDisplay }}</span>
        <span class="text-muted-foreground text-[10px]">{{ expanded ? '[-]' : '[+]' }}</span>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div v-if="toolCall.output" class="border-t border-border px-2 py-1.5">
          <pre class="whitespace-pre-wrap text-primary/70 font-mono max-h-48 overflow-y-auto text-[11px]">{{ toolCall.output }}</pre>
        </div>
      </CollapsibleContent>
      <div v-if="!toolCall.output && !expanded" class="px-2 py-1 text-primary/50">
        <span class="animate-pulse">executing...</span>
      </div>
    </div>
  </Collapsible>
</template>
