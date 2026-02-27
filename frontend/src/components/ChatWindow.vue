<script setup lang="ts">
import { watch, nextTick } from 'vue'
import { useChatStore } from '@/stores/chat'
import MessageBubble from './MessageBubble.vue'
import { ScrollArea } from '@/components/ui/scroll-area'

const chat = useChatStore()

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
      <MessageBubble
        v-for="(msg, i) in chat.messages"
        :key="i"
        :message="msg"
      />
      <div v-if="chat.error" class="text-sm text-destructive bg-destructive/10 rounded-md p-3">
        {{ chat.error }}
      </div>
    </div>
  </ScrollArea>
</template>
