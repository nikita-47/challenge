<script setup lang="ts">
import { ref, watch, nextTick, computed } from 'vue'
import { marked } from 'marked'
import { useSupportStore } from '@/stores/support'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'

marked.setOptions({ breaks: true })

const support = useSupportStore()
const input = ref('')
const scrollAnchor = ref<HTMLDivElement | null>(null)
const textareaRef = ref<HTMLTextAreaElement | null>(null)

const canSend = computed(() => input.value.trim().length > 0 && !support.sending)

function send() {
  const text = input.value.trim()
  if (!text || support.sending) {
    return
  }
  input.value = ''
  // Reset textarea height
  if (textareaRef.value) {
    textareaRef.value.style.height = 'auto'
  }
  support.sendMessage(text)
}

function handleTextareaInput(event: Event) {
  const el = event.target as HTMLTextAreaElement
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 80) + 'px'
}

function scrollToBottom() {
  nextTick(() => {
    scrollAnchor.value?.scrollIntoView({ behavior: 'smooth' })
  })
}

watch(
  () => support.messages,
  () => {
    scrollToBottom()
  },
  { deep: true },
)

function renderedContent(content: string): string {
  return marked.parse(content) as string
}
</script>

<template>
  <!-- Floating button (closed state) -->
  <button
    v-if="!support.isOpen"
    class="absolute bottom-16 right-4 z-[60] flex h-12 w-12 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-lg hover:bg-primary/90 transition-all hover:scale-105"
    title="Support"
    @click="support.toggle()"
  >
    <span class="text-lg font-bold">?</span>
  </button>

  <!-- Chat panel (open state) -->
  <div
    v-else
    class="absolute bottom-16 right-4 z-[60] flex h-[500px] w-96 flex-col rounded-lg border border-border bg-card shadow-xl"
  >
    <!-- Header -->
    <div class="flex items-center justify-between border-b border-border px-4 py-2">
      <span class="font-mono text-sm font-semibold text-foreground">Support</span>
      <div class="flex gap-1">
        <button
          class="flex h-6 w-6 items-center justify-center rounded text-xs text-muted-foreground hover:bg-muted/40 hover:text-foreground transition-colors"
          title="Clear chat"
          @click="support.clearChat()"
        >
          &#10005;&#10005;
        </button>
        <button
          class="flex h-6 w-6 items-center justify-center rounded text-xs text-muted-foreground hover:bg-muted/40 hover:text-foreground transition-colors"
          title="Close"
          @click="support.toggle()"
        >
          &#8722;
        </button>
      </div>
    </div>

    <!-- Messages area -->
    <ScrollArea class="flex-1 px-3 py-2">
      <!-- Empty state -->
      <div
        v-if="support.messages.length === 0"
        class="flex h-full items-center justify-center py-8 text-center"
      >
        <p class="font-mono text-xs text-muted-foreground">How can I help you?</p>
      </div>

      <!-- Messages -->
      <div class="flex flex-col gap-2">
        <div
          v-for="(msg, idx) in support.messages"
          :key="idx"
          :class="[
            'flex',
            msg.role === 'user' ? 'justify-end' : 'justify-start',
          ]"
        >
          <!-- User bubble -->
          <div
            v-if="msg.role === 'user'"
            class="max-w-[80%] rounded-lg bg-primary px-3 py-2 text-xs font-mono text-primary-foreground"
          >
            {{ msg.content }}
          </div>

          <!-- Assistant bubble -->
          <div
            v-else
            class="max-w-[80%] rounded-lg bg-muted px-3 py-2 text-xs font-mono text-foreground"
          >
            <div
              v-if="msg.content"
              class="prose prose-sm max-w-none [&_p]:mb-1 [&_p:last-child]:mb-0 [&_ul]:my-1 [&_ol]:my-1 [&_li]:my-0 [&_code]:bg-background/50 [&_code]:px-1 [&_code]:rounded [&_pre]:bg-background/50 [&_pre]:p-2 [&_pre]:rounded [&_pre]:overflow-x-auto"
              v-html="renderedContent(msg.content)"
            />
            <span
              v-if="msg.isStreaming"
              class="inline-flex gap-0.5 align-middle"
            >
              <span class="inline-block h-1 w-1 animate-bounce rounded-full bg-muted-foreground/60 [animation-delay:0ms]" />
              <span class="inline-block h-1 w-1 animate-bounce rounded-full bg-muted-foreground/60 [animation-delay:150ms]" />
              <span class="inline-block h-1 w-1 animate-bounce rounded-full bg-muted-foreground/60 [animation-delay:300ms]" />
            </span>
          </div>
        </div>
      </div>

      <div ref="scrollAnchor" />
    </ScrollArea>

    <!-- Error bar -->
    <div
      v-if="support.error"
      class="border-t border-red-500/20 bg-red-500/10 px-3 py-1.5 text-xs font-mono text-red-400"
    >
      {{ support.error }}
    </div>

    <!-- Input area -->
    <div class="border-t border-border p-2">
      <div class="flex gap-2">
        <textarea
          ref="textareaRef"
          v-model="input"
          class="flex-1 resize-none rounded border border-input bg-background px-3 py-2 text-xs font-mono placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
          rows="1"
          placeholder="Ask a question..."
          @keydown.enter.exact.prevent="send"
          @input="handleTextareaInput"
        />
        <Button
          size="sm"
          class="shrink-0 text-xs"
          :disabled="!canSend"
          @click="send"
        >
          Send
        </Button>
      </div>
    </div>
  </div>
</template>
