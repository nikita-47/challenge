<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { marked } from 'marked'
import { useReviewStore } from '@/stores/review'
import { useUIStore } from '@/stores/ui'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import type { ReviewStepName, ReviewStepStatus } from '@/lib/types'

const review = useReviewStore()
const ui = useUIStore()

const STEP_ORDER: ReviewStepName[] = ['diff', 'rag', 'analyze', 'comment']

function stepState(name: ReviewStepName): ReviewStepStatus | 'pending' {
  const found = review.reviewSteps.find((s) => s.step === name)
  if (!found) {
    return 'pending'
  }
  return found.status
}

function stepDetail(name: ReviewStepName): string | undefined {
  return review.reviewSteps.find((s) => s.step === name)?.detail
}

function stepBadgeClass(status: ReviewStepStatus | 'pending'): string {
  switch (status) {
    case 'running': {
      return 'text-amber-400 border-amber-400/30 bg-amber-400/10 animate-pulse'
    }
    case 'done': {
      return 'text-emerald-400 border-emerald-400/30 bg-emerald-400/10'
    }
    case 'skipped': {
      return 'text-muted-foreground/50 border-muted-foreground/20 bg-transparent'
    }
    case 'error': {
      return 'text-red-400 border-red-400/30 bg-red-400/10'
    }
    default: {
      return 'text-muted-foreground/30 border-muted-foreground/10 bg-transparent'
    }
  }
}

const renderedMarkdown = computed(() => {
  if (!review.reviewText) {
    return ''
  }
  return marked.parse(review.reviewText) as string
})

onMounted(() => {
  review.loadPRs()
})
</script>

<template>
  <div class="flex flex-col h-screen w-screen bg-background font-mono overflow-hidden">
    <!-- Header bar -->
    <div class="flex items-center gap-4 px-4 py-2 border-b border-primary/20 bg-card shrink-0">
      <div class="text-sm text-primary font-semibold tracking-widest shrink-0">
        // code review
      </div>

      <div class="flex flex-1 items-center gap-2">
        <Button
          size="sm"
          variant="outline"
          class="h-7 text-xs shrink-0"
          :disabled="review.loading"
          @click="review.loadPRs()"
        >
          {{ review.loading ? 'loading...' : '↺ refresh' }}
        </Button>
      </div>

      <Button
        variant="ghost"
        size="sm"
        class="h-7 text-xs text-muted-foreground hover:text-foreground shrink-0"
        @click="ui.setView('chat')"
      >
        ← chat
      </Button>
    </div>

    <!-- Error bar -->
    <div
      v-if="review.error"
      class="shrink-0 text-xs text-red-400 border-b border-red-400/30 px-4 py-1.5 bg-red-400/5 font-mono"
    >
      {{ review.error }}
    </div>

    <!-- Body -->
    <div class="flex flex-1 min-h-0">
      <!-- Left column: PR list -->
      <div class="w-72 shrink-0 border-r border-primary/20 flex flex-col">
        <div class="flex items-center justify-between px-3 py-2 border-b border-primary/10">
          <span class="text-[10px] text-primary/60 uppercase tracking-wider">
            // open prs ({{ review.prs.length }})
          </span>
        </div>

        <ScrollArea class="flex-1">
          <div class="p-2 space-y-1">
            <div
              v-if="review.prs.length === 0 && !review.loading"
              class="text-xs text-muted-foreground/50 font-mono py-3 px-1"
            >
              // no open pull requests
            </div>
            <div
              v-for="pr in review.prs"
              :key="pr.number"
              class="group flex items-start gap-2 px-2 py-2 border cursor-pointer transition-colors"
              :class="review.selectedPR?.number === pr.number
                ? 'border-primary/40 bg-primary/5 text-primary'
                : 'border-transparent hover:border-primary/20 hover:bg-primary/5'"
              @click="review.selectPR(pr)"
            >
              <div class="min-w-0 flex-1 space-y-1">
                <div class="text-xs font-mono text-foreground truncate">
                  #{{ pr.number }} {{ pr.title }}
                </div>
                <div class="flex items-center gap-2">
                  <span class="text-[10px] text-muted-foreground/60">{{ pr.author }}</span>
                  <span class="text-[10px] text-primary/50">{{ pr.branch }}</span>
                </div>
                <div v-if="pr.labels.length > 0" class="flex flex-wrap gap-1">
                  <Badge
                    v-for="label in pr.labels"
                    :key="label"
                    variant="outline"
                    class="text-[9px] px-1 py-0 text-muted-foreground/60 border-muted-foreground/20"
                  >
                    {{ label }}
                  </Badge>
                </div>
              </div>
            </div>
          </div>
        </ScrollArea>
      </div>

      <!-- Center area -->
      <div class="flex-1 flex flex-col min-h-0">
        <!-- Empty state -->
        <div
          v-if="!review.selectedPR"
          class="flex-1 flex items-center justify-center text-muted-foreground/30 text-sm font-mono"
        >
          // select a pull request
        </div>

        <!-- PR detail -->
        <template v-else>
          <!-- Meta bar -->
          <div class="shrink-0 px-4 py-2.5 border-b border-primary/10 flex items-center gap-4 flex-wrap text-[11px] font-mono text-muted-foreground">
            <span class="text-foreground font-semibold truncate">{{ review.selectedPR.title }}</span>
            <span class="text-primary/70 shrink-0">#{{ review.selectedPR.number }}</span>
            <span class="shrink-0">{{ review.selectedPR.author }}</span>
            <span class="text-primary/50 shrink-0">{{ review.selectedPR.branch }}</span>
            <a
              v-if="review.selectedPR.url"
              :href="review.selectedPR.url"
              target="_blank"
              rel="noopener noreferrer"
              class="text-primary/50 hover:text-primary transition-colors shrink-0"
            >
              ↗ open
            </a>
          </div>

          <!-- Button row -->
          <div class="shrink-0 px-4 py-2 border-b border-primary/10 flex items-center gap-2">
            <Button
              size="sm"
              class="h-7 px-4 text-xs"
              :disabled="review.reviewing"
              @click="review.runReview()"
            >
              {{ review.reviewing ? 'reviewing...' : 'run review' }}
            </Button>
            <Button
              v-if="review.reviewing"
              size="sm"
              variant="outline"
              class="h-7 px-3 text-xs text-red-400 border-red-400/30 hover:bg-red-400/10"
              @click="review.cancelReview()"
            >
              cancel
            </Button>
          </div>

          <!-- Steps indicator -->
          <div
            v-if="review.reviewSteps.length > 0 || review.reviewing"
            class="shrink-0 px-4 py-2 border-b border-primary/10 flex items-center gap-4 flex-wrap"
          >
            <div
              v-for="stepName in STEP_ORDER"
              :key="stepName"
              class="flex items-center gap-1.5"
            >
              <span class="text-[10px] text-muted-foreground/60 font-mono">{{ stepName }}</span>
              <Badge
                variant="outline"
                class="text-[9px] px-1.5 py-0"
                :class="stepBadgeClass(stepState(stepName))"
              >
                {{ stepState(stepName) }}
              </Badge>
              <span
                v-if="stepDetail(stepName)"
                class="text-[9px] text-muted-foreground/40 font-mono truncate max-w-32"
              >
                {{ stepDetail(stepName) }}
              </span>
            </div>
          </div>

          <!-- Review result -->
          <ScrollArea class="flex-1">
            <div
              v-if="!review.reviewText"
              class="flex items-center justify-center h-32 text-muted-foreground/30 text-xs font-mono"
            >
              <span v-if="review.reviewing" class="animate-pulse">// analyzing...</span>
              <span v-else>// run review to see results</span>
            </div>
            <div
              v-else
              class="p-4 prose prose-invert prose-sm max-w-none"
              v-html="renderedMarkdown"
            />
          </ScrollArea>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.prose :deep(h1),
.prose :deep(h2),
.prose :deep(h3),
.prose :deep(h4) {
  font-family: ui-monospace, monospace;
  color: hsl(var(--primary));
}

.prose :deep(code) {
  font-family: ui-monospace, monospace;
  font-size: 0.8em;
  background-color: hsl(var(--muted) / 0.5);
  padding: 0.1em 0.3em;
  border-radius: 0.2em;
}

.prose :deep(pre) {
  background-color: hsl(var(--muted) / 0.3);
  border: 1px solid hsl(var(--primary) / 0.1);
  border-radius: 0.25rem;
}

.prose :deep(pre code) {
  background-color: transparent;
  padding: 0;
}

.prose :deep(blockquote) {
  border-left-color: hsl(var(--primary) / 0.3);
  color: hsl(var(--muted-foreground));
}

.prose :deep(a) {
  color: hsl(var(--primary));
}

.prose :deep(hr) {
  border-color: hsl(var(--primary) / 0.1);
}
</style>
