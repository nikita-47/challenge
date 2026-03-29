import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchConfig, fetchSettings, updateSettings, type AppConfig, type ProviderSettings } from '@/lib/api'

export const useUIStore = defineStore('ui', () => {
  const leftSidebarOpen = ref(true)
  const rightSidebarOpen = ref(false)
  const config = ref<AppConfig | null>(null)
  const newChatDialogOpen = ref(false)
  const activeView = ref<'chat' | 'pipeline' | 'docs'>('chat')
  const providerSettings = ref<ProviderSettings>({
    provider: 'local',
    localURL: 'http://localhost:1234',
    localModel: 'qwen2.5-0.5b-instruct-mlx',
    localKey: '',
  })

  function setView(view: 'chat' | 'pipeline' | 'docs') {
    activeView.value = view
  }

  function toggleLeftSidebar() {
    leftSidebarOpen.value = !leftSidebarOpen.value
  }

  function toggleRightSidebar() {
    rightSidebarOpen.value = !rightSidebarOpen.value
  }

  async function loadConfig() {
    try {
      config.value = await fetchConfig()
    } catch {
      config.value = null
    }
  }

  async function loadSettings() {
    try {
      providerSettings.value = await fetchSettings()
    } catch {
      // keep defaults
    }
  }

  async function saveSettings(settings: ProviderSettings) {
    providerSettings.value = await updateSettings(settings)
  }

  return {
    leftSidebarOpen,
    rightSidebarOpen,
    config,
    newChatDialogOpen,
    providerSettings,
    activeView,
    toggleLeftSidebar,
    toggleRightSidebar,
    loadConfig,
    loadSettings,
    saveSettings,
    setView,
  }
})
