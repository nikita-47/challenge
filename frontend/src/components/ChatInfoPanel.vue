<script setup lang="ts">
import { computed } from 'vue'
import { useChatStore } from '@/stores/chat'
import { useUIStore } from '@/stores/ui'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Button } from '@/components/ui/button'

defineEmits<{ close: [] }>()

const chat = useChatStore()
const ui = useUIStore()

const displayConfig = computed(() => {
  if (chat.settings) {
    return {
      model: chat.settings.model,
      maxTokens: chat.settings.maxTokens,
      temperature: chat.settings.temperature,
      system: chat.settings.system,
    }
  }
  return ui.config
})
</script>

<template>
  <aside class="flex flex-col border-l border-border bg-muted h-full">
    <div class="p-3 border-b border-border flex items-center justify-between">
      <h2 class="text-sm font-semibold text-foreground">Chat Info</h2>
      <Button
        variant="ghost"
        size="icon"
        class="h-6 w-6"
        @click="$emit('close')"
      >
        &times;
      </Button>
    </div>
    <ScrollArea class="flex-1">
      <div class="p-3 space-y-3 text-sm">
        <!-- Session -->
        <div class="space-y-2">
          <div class="text-xs font-medium text-muted-foreground uppercase tracking-wide">
            Session
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">Name</span>
            <span class="text-foreground truncate ml-2 max-w-[140px]" :title="chat.currentSession">
              {{ chat.currentSession }}
            </span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">Exchanges</span>
            <span class="text-foreground">{{ chat.exchanges }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">Tokens in</span>
            <span class="text-foreground">{{ chat.totalUsage.input.toLocaleString() }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">Tokens out</span>
            <span class="text-foreground">{{ chat.totalUsage.output.toLocaleString() }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">Cost</span>
            <span class="text-foreground">${{ chat.totalCost.toFixed(6) }}</span>
          </div>
          <div v-if="chat.hasSummary" class="flex justify-between">
            <span class="text-muted-foreground">Compressed</span>
            <span class="text-foreground">
              <template v-if="chat.compressionCount > 0">
                {{ chat.compressionCount }}x
              </template>
              <template v-else>Yes</template>
              <span v-if="chat.tokensSaved > 0" class="text-muted-foreground">
                (~{{ chat.tokensSaved.toLocaleString() }} saved)
              </span>
            </span>
          </div>
        </div>

        <Separator />

        <!-- Config -->
        <div class="space-y-2">
          <div class="text-xs font-medium text-muted-foreground uppercase tracking-wide">
            Config
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">Model</span>
            <span class="text-foreground text-xs truncate ml-2 max-w-[140px]" :title="displayConfig?.model">
              {{ displayConfig?.model ?? '—' }}
            </span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">Max tokens</span>
            <span class="text-foreground">{{ displayConfig?.maxTokens ?? '—' }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">Temperature</span>
            <span class="text-foreground">
              {{ displayConfig?.temperature === -1 ? 'default' : displayConfig?.temperature ?? '—' }}
            </span>
          </div>
          <div v-if="displayConfig?.system" class="space-y-1">
            <span class="text-muted-foreground">System prompt</span>
            <p class="text-foreground text-xs bg-background rounded p-2 break-words">
              {{ displayConfig.system }}
            </p>
          </div>
          <div v-else class="flex justify-between">
            <span class="text-muted-foreground">System prompt</span>
            <span class="text-foreground">—</span>
          </div>
        </div>
      </div>
    </ScrollArea>
  </aside>
</template>
