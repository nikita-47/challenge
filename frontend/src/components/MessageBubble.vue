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
      class="max-w-[85%] px-4 py-2"
      :class="isUser
        ? 'bg-card border-l-2 border-primary/60'
        : 'bg-card border-l-2 border-accent/60'"
    >
      <div
        class="text-xs font-medium mb-1"
        :class="isUser ? 'text-primary' : 'text-accent-foreground'"
      >
        {{ isUser ? '> you' : '$ claude' }}
      </div>
      <div
        class="prose prose-sm max-w-none break-words text-foreground [&_pre]:bg-background [&_pre]:border [&_pre]:border-border [&_pre]:p-2 [&_pre]:overflow-x-auto [&_code]:text-primary [&_code]:text-xs [&_a]:text-accent-foreground [&_a]:underline [&_strong]:text-foreground [&_h1]:text-foreground [&_h2]:text-foreground [&_h3]:text-foreground [&_h4]:text-foreground [&_li]:text-foreground [&_p]:text-foreground [&_blockquote]:border-primary/30 [&_blockquote]:text-muted-foreground [&_hr]:border-border"
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
        <span class="inline-block w-2 h-4 bg-primary animate-pulse" />
      </div>
    </div>
  </div>
</template>
