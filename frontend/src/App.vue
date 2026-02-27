<script setup lang="ts">
import { onMounted } from 'vue'
import SessionPanel from '@/components/SessionPanel.vue'
import ChatWindow from '@/components/ChatWindow.vue'
import ChatInput from '@/components/ChatInput.vue'
import TokenBar from '@/components/TokenBar.vue'
import { useSessionsStore } from '@/stores/sessions'

const sessionsStore = useSessionsStore()

onMounted(() => {
  sessionsStore.loadList()
})
</script>

<template>
  <div class="flex h-screen w-screen overflow-hidden">
    <SessionPanel class="w-64 shrink-0" />
    <div class="flex flex-1 flex-col min-w-0">
      <ChatWindow class="flex-1 overflow-hidden" />
      <TokenBar />
      <ChatInput />
    </div>
  </div>
</template>
