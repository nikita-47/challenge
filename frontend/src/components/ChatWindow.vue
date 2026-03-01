<script setup lang="ts">
import { watch, nextTick } from 'vue'
import { useChatStore } from '@/stores/chat'
import type { ChatMessage } from '@/lib/types'
import MessageBubble from './MessageBubble.vue'
import { ScrollArea } from '@/components/ui/scroll-area'

const chat = useChatStore()

function formatSystemEvent(msg: ChatMessage): string {
  if (!msg.event) {
    return msg.content
  }
  switch (msg.event.type) {
    case 'compress':
      return `compressed ${msg.event.messageCount} messages · saved ~${msg.event.tokensSaved} tokens`
    default:
      return msg.event.type
  }
}

function scrollToBottom() {
  nextTick(() => {
    const viewport = document.querySelector('[data-radix-scroll-area-viewport]')
    if (viewport) {
      viewport.scrollTop = viewport.scrollHeight
    }
  })
}

watch(
  () => chat.messages.length,
  scrollToBottom,
)

watch(
  () => chat.messages[chat.messages.length - 1]?.content,
  scrollToBottom,
)
</script>

<template>
  <ScrollArea class="flex-1">
    <div class="p-4 space-y-4">
      <div
        v-if="chat.messages.length === 0"
        class="flex items-center justify-center h-full min-h-[200px] text-muted-foreground text-sm"
      >
        Start a conversation...
      </div>
      <template v-for="(msg, i) in chat.messages" :key="i">
        <div
          v-if="msg.role === 'system'"
          class="flex items-center gap-3 text-xs text-muted-foreground"
        >
          <div class="flex-1 border-t border-border" />
          <span>{{ formatSystemEvent(msg) }}</span>
          <div class="flex-1 border-t border-border" />
        </div>
        <MessageBubble v-else :message="msg" />
      </template>
      <div v-if="chat.error" class="text-sm text-destructive bg-destructive/10 rounded-md p-3">
        {{ chat.error }}
      </div>
    </div>
  </ScrollArea>
</template>
