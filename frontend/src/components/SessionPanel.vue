<script setup lang="ts">
import { useSessionsStore } from '@/stores/sessions'
import { useChatStore } from '@/stores/chat'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'

const sessions = useSessionsStore()
const chat = useChatStore()
</script>

<template>
  <aside class="flex flex-col border-r border-border bg-muted h-full">
    <div class="p-3 border-b border-border">
      <h2 class="text-sm font-semibold text-foreground mb-2">Sessions</h2>
      <Button
        class="w-full"
        size="sm"
        @click="sessions.newChat()"
      >
        + New Chat
      </Button>
    </div>
    <ScrollArea class="flex-1">
      <div class="p-2 space-y-1">
        <div v-if="sessions.loading" class="text-xs text-muted-foreground p-2">
          Loading...
        </div>
        <div
          v-for="name in sessions.sessions"
          :key="name"
          class="group flex items-center justify-between px-2 py-1.5 rounded-md text-sm cursor-pointer transition-colors"
          :class="chat.currentSession === name ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:bg-background'"
          @click="sessions.loadSession(name)"
        >
          <span class="truncate">{{ name }}</span>
          <Button
            variant="ghost"
            size="icon"
            class="opacity-0 group-hover:opacity-100 h-6 w-6 text-destructive hover:text-destructive"
            @click.stop="sessions.deleteSession(name)"
            title="Delete session"
          >
            &times;
          </Button>
        </div>
        <div v-if="!sessions.loading && sessions.sessions.length === 0" class="text-xs text-muted-foreground p-2">
          No saved sessions
        </div>
      </div>
    </ScrollArea>
  </aside>
</template>
