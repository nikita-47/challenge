<script setup lang="ts">
import { ref } from 'vue'
import { useChatStore } from '@/stores/chat'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'

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
  <div class="border-t border-border p-3 bg-background">
    <div class="flex items-end gap-2">
      <div class="flex-1 relative">
        <textarea
          ref="textarea"
          v-model="input"
          @keydown="onKeydown"
          @input="autoGrow"
          :placeholder="chat.isAgentMode ? 'Describe a task for the agent...' : 'Type a message...'"
          class="flex w-full resize-none rounded-lg border border-input bg-muted text-foreground px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
          rows="1"
          :disabled="chat.isStreaming"
        />
      </div>
      <div class="flex items-center gap-2">
        <label class="flex items-center gap-1.5 text-xs text-muted-foreground cursor-pointer select-none">
          <Checkbox
            :checked="chat.isAgentMode"
            @update:checked="chat.isAgentMode = $event"
          />
          Agent
        </label>
        <Button
          v-if="chat.isStreaming"
          variant="destructive"
          size="sm"
          @click="chat.stopStreaming()"
        >
          Stop
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
