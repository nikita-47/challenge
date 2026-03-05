<script setup lang="ts">
import { computed } from 'vue'
import { useChatStore } from '@/stores/chat'
import { Button } from '@/components/ui/button'

const chat = useChatStore()

const phases = ['planning', 'executing', 'validating', 'done'] as const

const completedSteps = computed(() => {
  if (!chat.taskState) {
    return 0
  }
  return chat.taskState.steps.filter((s) => s.status === 'completed').length
})

const totalSteps = computed(() => chat.taskState?.steps.length ?? 0)

const nextPhaseLabel = computed(() => {
  if (!chat.taskState) {
    return ''
  }
  switch (chat.taskState.phase) {
    case 'planning':
      return 'planning'
    case 'executing':
      return 'executing'
    case 'validating':
      return 'validating'
    default:
      return ''
  }
})

function stepIcon(status: string) {
  switch (status) {
    case 'completed':
      return 'x'
    case 'failed':
      return '!'
    default:
      return '\u00A0'
  }
}

function stepClass(status: string) {
  switch (status) {
    case 'completed':
      return 'text-green-400 border-green-400/30'
    case 'failed':
      return 'text-red-400 border-red-400/30'
    default:
      return 'text-muted-foreground border-border'
  }
}
</script>

<template>
  <div
    v-if="chat.taskState && chat.taskState.phase !== 'done'"
    class="border-b border-primary/20 bg-card/50 px-3 py-2 space-y-2"
  >
    <!-- Phase indicators -->
    <div class="flex items-center gap-1">
      <span class="text-xs text-muted-foreground mr-1">task</span>
      <div
        v-for="phase in phases"
        :key="phase"
        class="flex items-center gap-1"
      >
        <span
          class="text-[10px] px-1.5 py-0.5 border transition-colors"
          :class="
            chat.taskState?.phase === phase
              ? chat.taskState?.paused
                ? 'bg-yellow-400/20 text-yellow-400 border-yellow-400/40'
                : 'bg-primary/20 text-primary border-primary/40 animate-pulse'
              : phases.indexOf(phase) < phases.indexOf(chat.taskState?.phase as typeof phases[number])
                ? 'bg-green-400/10 text-green-400 border-green-400/30'
                : 'text-muted-foreground border-border'
          "
        >
          {{ phase }}
        </span>
        <span v-if="phase !== 'done'" class="text-muted-foreground text-[10px]">&rarr;</span>
      </div>
      <div class="flex-1" />
      <span v-if="totalSteps > 0" class="text-xs text-muted-foreground">
        {{ completedSteps }}/{{ totalSteps }}
      </span>
      <!-- Continue button -->
      <Button
        v-if="chat.taskState?.paused && !chat.isStreaming"
        variant="ghost"
        size="sm"
        class="h-5 px-1.5 text-[10px] text-green-400 hover:text-green-300"
        @click="chat.continueTask()"
      >
        continue {{ nextPhaseLabel }}
      </Button>
      <!-- Cancel button -->
      <Button
        v-if="!chat.isStreaming"
        variant="ghost"
        size="sm"
        class="h-5 px-1.5 text-[10px] text-red-400 hover:text-red-300"
        @click="chat.cancelTask()"
      >
        cancel
      </Button>
    </div>

    <!-- Steps list -->
    <div v-if="chat.taskState && chat.taskState.steps.length > 0" class="space-y-0.5">
      <div
        v-for="step in chat.taskState.steps"
        :key="step.index"
        class="flex items-start gap-1.5 text-xs"
      >
        <span
          class="font-mono w-4 h-4 flex items-center justify-center border shrink-0 text-[10px]"
          :class="stepClass(step.status)"
        >
          {{ stepIcon(step.status) }}
        </span>
        <span
          class="break-words"
          :class="step.status === 'completed' ? 'text-muted-foreground line-through' : 'text-foreground'"
        >
          {{ step.description }}
        </span>
      </div>
    </div>

    <!-- Plan summary artifact -->
    <div
      v-if="chat.taskState?.artifacts?.plan_summary"
      class="text-[10px] text-muted-foreground bg-background/50 border border-border p-1.5"
    >
      <span class="text-primary">plan:</span> {{ chat.taskState.artifacts.plan_summary }}
    </div>

    <!-- Feedback from failed validation -->
    <div
      v-if="chat.taskState?.feedback"
      class="text-[10px] text-yellow-400 bg-yellow-400/5 border border-yellow-400/20 p-1.5"
    >
      <span class="font-medium">feedback:</span> {{ chat.taskState.feedback }}
    </div>

    <!-- Error -->
    <div
      v-if="chat.taskState?.error"
      class="text-[10px] text-red-400 bg-red-400/5 border border-red-400/20 p-1.5"
    >
      <span class="font-medium">error:</span> {{ chat.taskState.error }}
    </div>
  </div>
</template>
