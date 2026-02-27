<script setup lang="ts">
import { computed } from 'vue'
import type { ChatMessage } from '@/lib/types'
import { marked } from 'marked'
import ToolCallCard from './ToolCallCard.vue'

const props = defineProps<{
  message: ChatMessage
}>()

const renderedContent = computed(() => {
  if (!props.message.content) {
    return ''
  }
  return marked.parse(props.message.content, { async: false }) as string
})

const isUser = computed(() => props.message.role === 'user')
</script>

<template>
  <div class="flex" :class="isUser ? 'justify-end' : 'justify-start'">
    <div
      class="max-w-[80%] rounded-lg px-4 py-2"
      :class="isUser
        ? 'bg-primary text-primary-foreground'
        : 'bg-muted text-foreground'"
    >
      <div class="text-xs font-medium mb-1 opacity-70">
        {{ isUser ? 'You' : 'Claude' }}
      </div>
      <div
        class="prose prose-invert prose-sm max-w-none break-words [&_pre]:bg-black/30 [&_pre]:rounded [&_pre]:p-2 [&_pre]:overflow-x-auto [&_code]:text-amber-300 [&_code]:text-xs"
        v-html="renderedContent"
      />
      <div v-if="message.toolCalls?.length" class="mt-2 space-y-2">
        <ToolCallCard
          v-for="(tc, i) in message.toolCalls"
          :key="i"
          :toolCall="tc"
        />
      </div>
      <div v-if="message.isStreaming" class="mt-1">
        <span class="inline-block w-2 h-4 bg-primary animate-pulse rounded-sm" />
      </div>
    </div>
  </div>
</template>
