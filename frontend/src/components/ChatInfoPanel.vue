<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useChatStore } from '@/stores/chat'
import { useUIStore } from '@/stores/ui'
import { fetchSessionRaw } from '@/lib/api'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Button } from '@/components/ui/button'

defineEmits<{ close: [] }>()

const chat = useChatStore()
const ui = useUIStore()

const tab = ref<'info' | 'raw'>('info')
const rawJson = ref('')
const rawLoading = ref(false)

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

async function loadRaw() {
  if (!chat.currentSession) {
    return
  }
  rawLoading.value = true
  try {
    rawJson.value = await fetchSessionRaw(chat.currentSession)
  } catch {
    rawJson.value = '// failed to load'
  } finally {
    rawLoading.value = false
  }
}

watch(tab, (v) => {
  if (v === 'raw') {
    loadRaw()
  }
})

watch(() => chat.currentSession, () => {
  if (tab.value === 'raw') {
    loadRaw()
  }
})

watch(() => chat.messages.length, () => {
  if (tab.value === 'raw') {
    loadRaw()
  }
})
</script>

<template>
  <aside class="flex flex-col border-l border-border bg-muted h-full">
    <div class="p-3 border-b border-border flex items-center justify-between">
      <div class="flex items-center gap-1">
        <button
          class="px-2 py-0.5 text-xs rounded transition-colors"
          :class="tab === 'info' ? 'bg-background text-foreground font-medium shadow-sm' : 'text-muted-foreground hover:text-foreground'"
          @click="tab = 'info'"
        >
          Info
        </button>
        <button
          class="px-2 py-0.5 text-xs rounded transition-colors"
          :class="tab === 'raw' ? 'bg-background text-foreground font-medium shadow-sm' : 'text-muted-foreground hover:text-foreground'"
          @click="tab = 'raw'"
        >
          Raw
        </button>
      </div>
      <Button
        variant="ghost"
        size="icon"
        class="h-6 w-6"
        @click="$emit('close')"
      >
        &times;
      </Button>
    </div>

    <!-- Raw JSON tab -->
    <ScrollArea v-if="tab === 'raw'" class="flex-1">
      <pre class="p-3 text-xs text-foreground whitespace-pre-wrap break-words font-mono">{{ rawJson }}</pre>
    </ScrollArea>

    <!-- Info tab -->
    <ScrollArea v-else class="flex-1">
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

        <!-- Strategy -->
        <div v-if="chat.settings?.strategy" class="space-y-2">
          <Separator />
          <div class="text-xs font-medium text-muted-foreground uppercase tracking-wide">
            Strategy
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">Type</span>
            <span class="text-foreground capitalize">{{ chat.settings.strategy }}</span>
          </div>
          <div v-if="chat.settings.strategy === 'window' || chat.settings.strategy === 'facts'" class="flex justify-between">
            <span class="text-muted-foreground">Window size</span>
            <span class="text-foreground">{{ chat.settings.windowSize ?? 20 }}</span>
          </div>
          <div v-if="chat.settings.strategy === 'facts'" class="flex justify-between">
            <span class="text-muted-foreground">Facts</span>
            <span class="text-foreground">{{ Object.keys(chat.facts).length }} entries</span>
          </div>
          <!-- Facts detail -->
          <div v-if="chat.settings.strategy === 'facts' && Object.keys(chat.facts).length > 0" class="space-y-1">
            <div
              v-for="(val, key) in chat.facts"
              :key="key"
              class="text-xs bg-background rounded p-1.5 break-words"
            >
              <span class="text-muted-foreground">{{ key }}:</span>
              <span class="text-foreground ml-1">{{ val }}</span>
            </div>
          </div>
          <!-- Branch info -->
          <div v-if="chat.settings.strategy === 'branch'" class="flex justify-between">
            <span class="text-muted-foreground">Active branch</span>
            <span class="text-foreground">{{ chat.activeBranch || 'main' }}</span>
          </div>
          <div v-if="chat.settings.strategy === 'branch' && chat.branches.length > 0" class="space-y-1">
            <div class="text-xs text-muted-foreground">Branches:</div>
            <div class="flex justify-between text-xs" v-for="b in chat.branches" :key="b.name">
              <span :class="b.name === chat.activeBranch ? 'text-foreground font-medium' : 'text-muted-foreground'">
                {{ b.name }}
              </span>
              <span class="text-muted-foreground">{{ b.messageCount }} msgs</span>
            </div>
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
