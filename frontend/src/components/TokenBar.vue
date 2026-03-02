<script setup lang="ts">
import { useChatStore } from '@/stores/chat'
import { Badge } from '@/components/ui/badge'

const chat = useChatStore()
</script>

<template>
  <div
    v-if="chat.exchanges > 0"
    class="flex items-center gap-1 px-4 py-1 border-t border-primary/20 bg-card text-xs font-mono"
  >
    <Badge variant="default">
      {{ chat.totalUsage.input.toLocaleString() }} in
    </Badge>
    <span class="text-muted-foreground">|</span>
    <Badge variant="default">
      {{ chat.totalUsage.output.toLocaleString() }} out
    </Badge>
    <span class="text-muted-foreground">|</span>
    <Badge variant="secondary">
      {{ chat.exchanges }}x
    </Badge>
    <span class="text-muted-foreground">|</span>
    <Badge variant="secondary">
      ${{ chat.totalCost.toFixed(6) }}
    </Badge>
    <span class="ml-auto text-xs text-muted-foreground">
      {{ chat.currentSession }}
    </span>
  </div>
</template>
