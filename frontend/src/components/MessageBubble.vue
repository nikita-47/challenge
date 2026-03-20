<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ChatMessage } from '@/lib/types'
import { marked } from 'marked'
import ToolCallCard from './ToolCallCard.vue'
import RAGPipelineSteps from './RAGPipelineSteps.vue'
import RAGChunkSplitView from './RAGChunkSplitView.vue'
import RAGSourcesBlock from './RAGSourcesBlock.vue'
import { Collapsible, CollapsibleTrigger, CollapsibleContent } from '@/components/ui/collapsible'
import { renderCitations } from '@/lib/ragCitations'

const props = defineProps<{
  message: ChatMessage
}>()

const apiRequestExpanded = ref(false)
const ragContextExpanded = ref(false)

const renderedContent = computed(() => {
  if (!props.message.content) {
    return ''
  }
  let html = marked.parse(props.message.content, { async: false }) as string
  if (props.message.ragSources?.length) {
    html = renderCitations(html, props.message.ragSources)
  }
  return html
})

const isUser = computed(() => props.message.role === 'user')

const ragChunkCount = computed(() => props.message.ragContext?.length ?? 0)

const ragDocCount = computed(() => {
  if (!props.message.ragContext) {
    return 0
  }
  return new Set(props.message.ragContext.map((r) => r.doc_name)).size
})
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
      <RAGPipelineSteps
        v-if="!isUser && message.ragSteps && message.ragSteps.length > 0"
        :steps="message.ragSteps"
        :rewritten-query="message.ragRewrittenQuery"
      />

      <Collapsible
        v-if="!isUser && message.ragContext && message.ragContext.length > 0"
        v-model:open="ragContextExpanded"
        class="mb-2"
      >
        <div class="border border-border/50 bg-background/50 text-xs rounded-sm">
          <CollapsibleTrigger class="w-full flex items-center gap-2 px-2 py-1.5 text-left hover:bg-muted/30 transition-colors">
            <span class="text-muted-foreground font-mono text-[10px]">rag</span>
            <span class="text-muted-foreground/80">
              {{ ragChunkCount }} chunk{{ ragChunkCount !== 1 ? 's' : '' }} from {{ ragDocCount }} doc{{ ragDocCount !== 1 ? 's' : '' }}
            </span>
            <span class="text-muted-foreground/50 text-[10px] ml-auto">{{ ragContextExpanded ? '[-]' : '[+]' }}</span>
          </CollapsibleTrigger>
          <CollapsibleContent>
            <div class="border-t border-border/50">
              <!-- Split view when we have pre/post filter data -->
              <RAGChunkSplitView
                v-if="message.ragAllResults && message.ragAllResults.length > 0"
                :passed="message.ragContext || []"
                :rejected="message.ragRejected || []"
                :threshold="message.ragThreshold || 0"
              />
              <!-- Fallback: original flat list for older messages -->
              <template v-else>
                <div class="divide-y divide-border/30">
                  <div
                    v-for="(result, i) in message.ragContext"
                    :key="i"
                    class="px-2 py-1.5 flex flex-col gap-0.5"
                  >
                    <div class="flex items-center gap-2">
                      <span class="text-muted-foreground/70 font-medium truncate flex-1" :title="result.doc_name">
                        {{ result.doc_name }}
                      </span>
                      <span class="text-emerald-500/70 font-mono shrink-0">
                        {{ (result.score * 100).toFixed(0) }}%
                      </span>
                    </div>
                    <p class="text-muted-foreground/60 leading-snug">
                      {{ result.chunk.text.length > 150 ? result.chunk.text.slice(0, 150) + '…' : result.chunk.text }}
                    </p>
                  </div>
                </div>
              </template>
            </div>
          </CollapsibleContent>
        </div>
      </Collapsible>

      <div
        v-if="!isUser && message.ragNoContext"
        class="flex items-center gap-2 px-2 py-1.5 mb-2 border border-amber-500/30 bg-amber-500/5 rounded-sm text-xs text-amber-400"
      >
        <span class="text-amber-500">!</span>
        <span>No relevant documents found -- response based on anti-hallucination rules</span>
      </div>

      <div
        class="prose prose-sm max-w-none break-words text-foreground [&_pre]:bg-background [&_pre]:border [&_pre]:border-border [&_pre]:p-2 [&_pre]:overflow-x-auto [&_code]:text-primary [&_code]:text-xs [&_a]:text-accent-foreground [&_a]:underline [&_strong]:text-foreground [&_h1]:text-foreground [&_h2]:text-foreground [&_h3]:text-foreground [&_h4]:text-foreground [&_li]:text-foreground [&_p]:text-foreground [&_blockquote]:border-primary/30 [&_blockquote]:text-muted-foreground [&_hr]:border-border"
        v-html="renderedContent"
      />

      <RAGSourcesBlock
        v-if="!isUser && message.ragSources && message.ragSources.length > 0"
        :sources="message.ragSources"
      />
      <Collapsible v-if="isUser && message.apiRequest" v-model:open="apiRequestExpanded" class="mt-2">
        <div class="border border-border bg-background text-xs">
          <CollapsibleTrigger class="w-full flex items-center gap-2 px-2 py-1.5 text-left hover:bg-primary/5 transition-colors">
            <span class="text-primary font-mono">api_request</span>
            <span class="text-muted-foreground text-[10px] ml-auto">{{ apiRequestExpanded ? '[-]' : '[+]' }}</span>
          </CollapsibleTrigger>
          <CollapsibleContent>
            <div class="border-t border-border px-2 py-1.5">
              <pre class="whitespace-pre-wrap text-primary/70 font-mono max-h-96 overflow-y-auto text-[11px]">{{ message.apiRequest }}</pre>
            </div>
          </CollapsibleContent>
        </div>
      </Collapsible>
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

<style scoped>
:deep(.rag-cite) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1rem;
  height: 1rem;
  padding: 0 0.25rem;
  font-size: 10px;
  font-family: monospace;
  font-weight: 600;
  background: hsl(var(--primary) / 0.15);
  color: hsl(var(--primary));
  border-radius: 2px;
  cursor: help;
  margin: 0 1px;
  vertical-align: super;
  line-height: 1;
}

:deep(.rag-cite:hover) {
  background: hsl(var(--primary) / 0.25);
}
</style>
