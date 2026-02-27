<script setup lang="ts">
import { useChatStore } from '@/stores/chat'
import { Badge } from '@/components/ui/badge'

const chat = useChatStore()
</script>

<template>
  <div
    v-if="chat.exchanges > 0"
    class="flex items-center gap-2 px-4 py-1.5 border-t border-border bg-background"
  >
    <Badge variant="secondary">
      {{ chat.totalUsage.input.toLocaleString() }} in / {{ chat.totalUsage.output.toLocaleString() }} out
    </Badge>
    <Badge variant="secondary">
      {{ chat.exchanges }} exchange{{ chat.exchanges !== 1 ? 's' : '' }}
    </Badge>
    <Badge variant="secondary">
      ${{ chat.totalCost.toFixed(6) }}
    </Badge>
    <span class="ml-auto text-xs text-muted-foreground">
      Session: {{ chat.currentSession }}
    </span>
  </div>
</template>
