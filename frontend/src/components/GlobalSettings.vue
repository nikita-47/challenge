<script setup lang="ts">
import { ref, watch } from 'vue'
import { useUIStore } from '@/stores/ui'
import type { ProviderSettings } from '@/lib/api'

const ui = useUIStore()

const localProvider = ref<ProviderSettings['provider']>(ui.providerSettings.provider)
const localURL = ref(ui.providerSettings.localURL)
const localModel = ref(ui.providerSettings.localModel)
const saving = ref(false)
const saved = ref(false)

watch(
  () => ui.providerSettings,
  (v) => {
    localProvider.value = v.provider
    localURL.value = v.localURL
    localModel.value = v.localModel
  },
  { deep: true },
)

async function apply() {
  saving.value = true
  saved.value = false
  try {
    await ui.saveSettings({
      provider: localProvider.value,
      localURL: localURL.value,
      localModel: localModel.value,
    })
    saved.value = true
    setTimeout(() => {
      saved.value = false
    }, 1500)
  } catch (e) {
    console.error('Failed to save settings:', e)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="border-t border-primary/20 p-3 space-y-2">
    <div class="flex items-center gap-2">
      <div class="text-xs font-medium text-primary uppercase tracking-wider">
        // settings
      </div>
      <span
        class="inline-block w-1.5 h-1.5 rounded-full shrink-0"
        :class="localProvider === 'claude' ? 'bg-orange-400' : 'bg-green-400'"
        :title="localProvider === 'claude' ? 'Claude API' : 'Local LLM'"
      />
    </div>

    <div class="flex gap-1">
      <button
        class="flex-1 px-2 py-0.5 text-xs transition-colors border"
        :class="localProvider === 'claude' ? 'bg-primary/10 text-primary border-primary/30' : 'text-muted-foreground hover:text-foreground border-transparent'"
        @click="localProvider = 'claude'"
      >
        claude
      </button>
      <button
        class="flex-1 px-2 py-0.5 text-xs transition-colors border"
        :class="localProvider === 'local' ? 'bg-primary/10 text-primary border-primary/30' : 'text-muted-foreground hover:text-foreground border-transparent'"
        @click="localProvider = 'local'"
      >
        local
      </button>
    </div>

    <template v-if="localProvider === 'local'">
      <input
        v-model="localURL"
        class="bg-background border border-border text-sm px-2 py-1 w-full outline-none focus:border-primary/30"
        placeholder="http://localhost:1234"
      />
      <input
        v-model="localModel"
        class="bg-background border border-border text-sm px-2 py-1 w-full outline-none focus:border-primary/30"
        placeholder="model name"
      />
    </template>

    <button
      class="px-2 py-0.5 text-xs border transition-colors"
      :class="saved ? 'text-green-400 border-green-400/30 bg-green-400/5' : 'text-muted-foreground border-transparent hover:text-foreground hover:border-primary/20'"
      :disabled="saving"
      @click="apply"
    >
      {{ saved ? 'saved' : saving ? 'saving...' : 'apply' }}
    </button>
  </div>
</template>
