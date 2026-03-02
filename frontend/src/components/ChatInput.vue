<script setup lang="ts">
import { ref } from 'vue'
import { useChatStore } from '@/stores/chat'
import { Button } from '@/components/ui/button'

const chat = useChatStore()
const input = ref('')
const textarea = ref<HTMLTextAreaElement | null>(null)

function send() {
  const text = input.value.trim()
  if (!text || chat.isStreaming) {
    return
  }
  input.value = ''
  if (textarea.value) {
    textarea.value.style.height = 'auto'
  }
  chat.sendMessage(text)
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    send()
  }
}

function autoGrow(e: Event) {
  const el = e.target as HTMLTextAreaElement
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 200) + 'px'
}
</script>

<template>
  <div class="border-t border-primary/20 p-3 bg-card">
    <div class="flex items-end gap-2">
      <span class="text-primary text-sm font-bold pb-2 select-none">&gt;_</span>
      <div class="flex-1 relative">
        <textarea
          ref="textarea"
          v-model="input"
          @keydown="onKeydown"
          @input="autoGrow"
          placeholder="Enter command..."
          class="flex w-full resize-none border border-input bg-background text-foreground px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary/50 focus-visible:border-primary/40 [caret-color:hsl(150_60%_45%)]"
          rows="1"
          :disabled="chat.isStreaming"
        />
      </div>
      <div class="flex items-center gap-2">
        <Button
          v-if="chat.isStreaming"
          variant="destructive"
          size="sm"
          @click="chat.stopStreaming()"
        >
          ^C
        </Button>
        <Button
          v-else
          size="sm"
          :disabled="!input.trim()"
          @click="send"
        >
          Send
        </Button>
      </div>
    </div>
  </div>
</template>
