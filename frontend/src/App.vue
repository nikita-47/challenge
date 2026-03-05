<script setup lang="ts">
import { onMounted } from 'vue'
import NewChatDialog from '@/components/NewChatDialog.vue'
import SessionPanel from '@/components/SessionPanel.vue'
import ChatInfoPanel from '@/components/ChatInfoPanel.vue'
import ChatWindow from '@/components/ChatWindow.vue'
import ChatInput from '@/components/ChatInput.vue'
import TokenBar from '@/components/TokenBar.vue'
import BranchSelector from '@/components/BranchSelector.vue'
import TaskStatePanel from '@/components/TaskStatePanel.vue'
import { Button } from '@/components/ui/button'
import { useSessionsStore } from '@/stores/sessions'
import { useChatStore } from '@/stores/chat'
import { useUIStore } from '@/stores/ui'
import { useMemoryStore } from '@/stores/memory'

const chat = useChatStore()

const sessionsStore = useSessionsStore()
const ui = useUIStore()
const memory = useMemoryStore()

onMounted(() => {
  sessionsStore.loadList()
  ui.loadConfig()
  ui.loadSettings()
  memory.loadAll()
})
</script>

<template>
  <div class="flex h-screen w-screen overflow-hidden bg-background">
    <!-- Left sidebar -->
    <SessionPanel
      v-if="ui.leftSidebarOpen"
      class="w-64 shrink-0"
      @close="ui.toggleLeftSidebar()"
    />

    <!-- Main area -->
    <div class="flex flex-1 flex-col min-w-0">
      <!-- Toolbar -->
      <div class="flex items-center gap-1 px-2 py-1 border-b border-primary/20 bg-card">
        <Button
          variant="ghost"
          size="icon"
          class="h-7 w-7 text-primary"
          @click="ui.toggleLeftSidebar()"
          title="Toggle sessions"
        >
          &#9776;
        </Button>
        <BranchSelector v-if="chat.settings?.strategy === 'branch'" />
        <div class="flex-1" />
        <Button
          variant="ghost"
          size="icon"
          class="h-7 w-7 text-muted-foreground"
          @click="ui.toggleRightSidebar()"
          title="Toggle chat info"
        >
          &#9432;
        </Button>
      </div>

      <TaskStatePanel />
      <ChatWindow class="flex-1 overflow-hidden" />
      <TokenBar />
      <ChatInput />
    </div>

    <!-- Right sidebar -->
    <ChatInfoPanel
      v-if="ui.rightSidebarOpen"
      class="w-72 shrink-0"
      @close="ui.toggleRightSidebar()"
    />
  </div>
  <NewChatDialog />
</template>
