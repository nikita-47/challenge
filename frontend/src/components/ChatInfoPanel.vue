<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useChatStore } from '@/stores/chat'
import { useUIStore } from '@/stores/ui'
import { fetchSessionRaw } from '@/lib/api'
import { ScrollArea } from '@/components/ui/scroll-area'
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
  <aside class="flex flex-col border-l border-primary/20 bg-card h-full">
    <div class="p-3 border-b border-primary/20 flex items-center justify-between">
      <div class="flex items-center gap-1">
        <button
          class="px-2 py-0.5 text-xs transition-colors border"
          :class="tab === 'info' ? 'bg-primary/10 text-primary border-primary/30' : 'text-muted-foreground hover:text-foreground border-transparent'"
          @click="tab = 'info'"
        >
          info
        </button>
        <button
          class="px-2 py-0.5 text-xs transition-colors border"
          :class="tab === 'raw' ? 'bg-primary/10 text-primary border-primary/30' : 'text-muted-foreground hover:text-foreground border-transparent'"
          @click="tab = 'raw'"
        >
          raw
        </button>
      </div>
      <Button
        variant="ghost"
        size="icon"
        class="h-6 w-6 text-muted-foreground"
        @click="$emit('close')"
      >
        &times;
      </Button>
    </div>

    <!-- Raw JSON tab -->
    <ScrollArea v-if="tab === 'raw'" class="flex-1">
      <pre class="p-3 text-xs text-primary/70 whitespace-pre-wrap break-words font-mono">{{ rawJson }}</pre>
    </ScrollArea>

    <!-- Info tab -->
    <ScrollArea v-else class="flex-1">
      <div class="p-3 space-y-3 text-sm">
        <!-- Session -->
        <div class="space-y-2">
          <div class="text-xs font-medium text-primary uppercase tracking-wider">
            // session
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">name</span>
            <span class="text-foreground truncate ml-2 max-w-[140px]" :title="chat.currentSession">
              {{ chat.currentSession }}
            </span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">exchanges</span>
            <span class="text-primary">{{ chat.exchanges }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">tokens_in</span>
            <span class="text-primary">{{ chat.totalUsage.input.toLocaleString() }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">tokens_out</span>
            <span class="text-primary">{{ chat.totalUsage.output.toLocaleString() }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">cost</span>
            <span class="text-accent-foreground">${{ chat.totalCost.toFixed(6) }}</span>
          </div>
          <div v-if="chat.hasSummary" class="flex justify-between">
            <span class="text-muted-foreground">compressed</span>
            <span class="text-foreground">
              <template v-if="chat.compressionCount > 0">
                {{ chat.compressionCount }}x
              </template>
              <template v-else>yes</template>
              <span v-if="chat.tokensSaved > 0" class="text-muted-foreground">
                (~{{ chat.tokensSaved.toLocaleString() }} saved)
              </span>
            </span>
          </div>
        </div>

        <!-- Task -->
        <div v-if="chat.taskState" class="space-y-2">
          <div class="border-t border-primary/10 my-2" />
          <div class="text-xs font-medium text-primary uppercase tracking-wider">
            // task
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">phase</span>
            <span
              :class="{
                'text-yellow-400': chat.taskState.phase === 'planning',
                'text-blue-400': chat.taskState.phase === 'executing',
                'text-orange-400': chat.taskState.phase === 'validating',
                'text-green-400': chat.taskState.phase === 'done',
              }"
            >
              {{ chat.taskState.phase }}
            </span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">paused</span>
            <span :class="chat.taskState.paused ? 'text-yellow-400' : 'text-muted-foreground'">
              {{ chat.taskState.paused }}
            </span>
          </div>
          <div v-if="chat.taskState.validation_count > 0" class="flex justify-between">
            <span class="text-muted-foreground">validation_count</span>
            <span class="text-primary">{{ chat.taskState.validation_count }}</span>
          </div>
          <div v-if="chat.taskState.steps.length > 0" class="flex justify-between">
            <span class="text-muted-foreground">progress</span>
            <span class="text-primary">
              {{ chat.taskState.steps.filter(s => s.status === 'completed').length }}/{{ chat.taskState.steps.length }}
            </span>
          </div>
          <div v-if="chat.taskState.step_results.length > 0" class="flex justify-between">
            <span class="text-muted-foreground">step_results</span>
            <span class="text-primary">{{ chat.taskState.step_results.length }}</span>
          </div>
          <div v-if="chat.taskState.goal" class="space-y-1">
            <span class="text-muted-foreground">goal</span>
            <p class="text-foreground text-xs bg-background border border-border p-1.5 break-words">
              {{ chat.taskState.goal }}
            </p>
          </div>
        </div>

        <!-- Strategy -->
        <div v-if="chat.settings?.strategy" class="space-y-2">
          <div class="border-t border-primary/10 my-2" />
          <div class="text-xs font-medium text-primary uppercase tracking-wider">
            // strategy
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">type</span>
            <span class="text-accent-foreground">{{ chat.settings.strategy }}</span>
          </div>
          <div v-if="chat.settings.strategy !== 'branch'" class="flex justify-between">
            <span class="text-muted-foreground">window_size</span>
            <span class="text-primary">{{ chat.settings.windowSize ?? 10 }}</span>
          </div>
          <div v-if="chat.settings.strategy === 'facts'" class="flex justify-between">
            <span class="text-muted-foreground">facts</span>
            <span class="text-primary">{{ Object.keys(chat.facts).length }} entries</span>
          </div>
          <!-- Facts detail -->
          <div v-if="chat.settings.strategy === 'facts' && Object.keys(chat.facts).length > 0" class="space-y-1">
            <div
              v-for="(val, key) in chat.facts"
              :key="key"
              class="text-xs bg-background border border-border p-1.5 break-words"
            >
              <span class="text-accent-foreground">{{ key }}:</span>
              <span class="text-foreground ml-1">{{ val }}</span>
            </div>
          </div>
          <!-- Branch info -->
          <div v-if="chat.settings.strategy === 'branch'" class="flex justify-between">
            <span class="text-muted-foreground">active_branch</span>
            <span class="text-primary">{{ chat.activeBranch || 'main' }}</span>
          </div>
          <div v-if="chat.settings.strategy === 'branch' && chat.branches.length > 0" class="space-y-1">
            <div class="text-xs text-muted-foreground">branches:</div>
            <div class="flex justify-between text-xs" v-for="b in chat.branches" :key="b.name">
              <span :class="b.name === chat.activeBranch ? 'text-primary font-medium' : 'text-muted-foreground'">
                {{ b.name }}
              </span>
              <span class="text-muted-foreground">{{ b.messageCount }} msgs</span>
            </div>
          </div>
        </div>

        <!-- Memory -->
        <div v-if="chat.settings?.operator || chat.settings?.profile || chat.settings?.project" class="space-y-2">
          <div class="border-t border-primary/10 my-2" />
          <div class="text-xs font-medium text-primary uppercase tracking-wider">
            // memory
          </div>
          <div v-if="chat.settings?.operator" class="flex justify-between">
            <span class="text-muted-foreground">operator</span>
            <span class="text-yellow-400">{{ chat.settings.operator }}</span>
          </div>
          <div v-if="chat.settings.profile" class="flex justify-between">
            <span class="text-muted-foreground">profile</span>
            <span class="text-accent-foreground">{{ chat.settings.profile }}</span>
          </div>
          <div v-if="chat.settings.project" class="flex justify-between">
            <span class="text-muted-foreground">project</span>
            <span class="text-accent-foreground">{{ chat.settings.project }}</span>
          </div>
        </div>

        <div class="border-t border-primary/10 my-2" />

        <!-- Config -->
        <div class="space-y-2">
          <div class="text-xs font-medium text-primary uppercase tracking-wider">
            // config
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">model</span>
            <span class="text-foreground text-xs truncate ml-2 max-w-[140px]" :title="displayConfig?.model">
              {{ displayConfig?.model ?? '—' }}
            </span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">max_tokens</span>
            <span class="text-primary">{{ displayConfig?.maxTokens ?? '—' }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">temperature</span>
            <span class="text-primary">
              {{ displayConfig?.temperature === -1 ? 'default' : displayConfig?.temperature ?? '—' }}
            </span>
          </div>
          <div v-if="displayConfig?.system" class="space-y-1">
            <span class="text-muted-foreground">system_prompt</span>
            <p class="text-foreground text-xs bg-background border border-border p-2 break-words">
              {{ displayConfig.system }}
            </p>
          </div>
          <div v-else class="flex justify-between">
            <span class="text-muted-foreground">system_prompt</span>
            <span class="text-foreground">—</span>
          </div>
        </div>
      </div>
    </ScrollArea>
  </aside>
</template>
