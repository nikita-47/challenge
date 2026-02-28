import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchConfig, type AppConfig } from '@/lib/api'

export const useUIStore = defineStore('ui', () => {
  const leftSidebarOpen = ref(true)
  const rightSidebarOpen = ref(false)
  const config = ref<AppConfig | null>(null)
  const newChatDialogOpen = ref(false)

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

  return {
    leftSidebarOpen,
    rightSidebarOpen,
    config,
    newChatDialogOpen,
    toggleLeftSidebar,
    toggleRightSidebar,
    loadConfig,
  }
})
