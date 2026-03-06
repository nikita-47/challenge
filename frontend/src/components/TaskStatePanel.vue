<script setup lang="ts">
import { computed } from 'vue'
import { useChatStore } from '@/stores/chat'
import { Button } from '@/components/ui/button'

const chat = useChatStore()

const hasPhases = computed(() => (chat.taskState?.phases?.length ?? 0) > 0)

const completedSteps = computed(() => {
  if (!chat.taskState?.steps) {
    return 0
  }
  return chat.taskState.steps.filter((s) => s.status === 'completed').length
})

const totalSteps = computed(() => chat.taskState?.steps?.length ?? 0)

const isProposingWithPhases = computed(() => {
  return chat.taskState?.phase === 'proposing' && hasPhases.value && chat.taskState?.paused
})

const continueLabel = computed(() => {
  if (!chat.taskState) {
    return ''
  }
  if (isProposingWithPhases.value) {
    return 'approve'
  }
  return chat.taskState.phase
})

function phaseStatusClass(status: string) {
  switch (status) {
    case 'active':
      return 'bg-primary/20 text-primary border-primary/40 animate-pulse'
    case 'completed':
      return 'bg-green-400/10 text-green-400 border-green-400/30'
    case 'failed':
      return 'bg-red-400/10 text-red-400 border-red-400/30'
    default:
      return 'text-muted-foreground border-border'
  }
}

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
    <div class="flex items-center gap-1 flex-wrap">
      <span class="text-xs text-muted-foreground mr-1">task</span>

      <!-- Proposing state: no phases yet -->
      <template v-if="chat.taskState.phase === 'proposing' && !hasPhases">
        <span class="text-[10px] px-1.5 py-0.5 border bg-primary/20 text-primary border-primary/40 animate-pulse">
          analyzing...
        </span>
      </template>

      <!-- Dynamic phases from pipeline -->
      <template v-else-if="hasPhases">
        <div
          v-for="(phase, index) in chat.taskState.phases"
          :key="phase.name"
          class="flex items-center gap-1"
        >
          <span
            class="text-[10px] px-1.5 py-0.5 border transition-colors"
            :class="
              isProposingWithPhases
                ? 'bg-yellow-400/20 text-yellow-400 border-yellow-400/40'
                : phaseStatusClass(phase.status)
            "
            :title="phase.description"
          >
            {{ phase.name }}
          </span>
          <span
            v-if="index < (chat.taskState.phases?.length ?? 0) - 1"
            class="text-muted-foreground text-[10px]"
          >&rarr;</span>
        </div>
      </template>

      <div class="flex-1" />
      <span v-if="totalSteps > 0" class="text-xs text-muted-foreground">
        {{ completedSteps }}/{{ totalSteps }}
      </span>
      <!-- Continue/Approve button -->
      <Button
        v-if="chat.taskState?.paused && !chat.isStreaming"
        variant="ghost"
        size="sm"
        class="h-5 px-1.5 text-[10px] text-green-400 hover:text-green-300"
        @click="chat.continueTask()"
      >
        {{ continueLabel }}
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

    <!-- Proposed phases detail (when awaiting approval) -->
    <div v-if="isProposingWithPhases" class="space-y-0.5">
      <p class="text-[10px] text-yellow-400/70 font-medium">proposed pipeline</p>
      <div
        v-for="(phase, index) in chat.taskState.phases"
        :key="phase.name"
        class="flex items-start gap-1.5 text-xs"
      >
        <span class="text-yellow-400/50 shrink-0 text-[10px] font-mono">{{ index + 1 }}.</span>
        <span class="text-foreground">
          <span class="text-primary font-medium">{{ phase.name }}</span>
          <span class="text-muted-foreground ml-1">[{{ phase.type }}]</span>
          <span class="text-muted-foreground ml-1">— {{ phase.description }}</span>
        </span>
      </div>
      <p class="text-[10px] text-muted-foreground mt-1">
        click approve or type feedback to modify
      </p>
    </div>

    <!-- Pipeline summary -->
    <div
      v-if="chat.taskState?.artifacts?.pipeline_summary && !isProposingWithPhases"
      class="text-[10px] text-muted-foreground bg-background/50 border border-border p-1.5"
    >
      <span class="text-primary">pipeline:</span> {{ chat.taskState.artifacts.pipeline_summary }}
    </div>

    <!-- Invariants -->
    <div v-if="chat.taskState?.invariants?.length" class="space-y-0.5">
      <p class="text-[10px] text-red-400/70 font-medium">invariants</p>
      <div
        v-for="(inv, index) in chat.taskState.invariants"
        :key="index"
        class="flex items-start gap-1.5 text-xs"
      >
        <span class="text-red-400/50 shrink-0 text-[10px]">!</span>
        <span class="text-muted-foreground break-words">{{ inv }}</span>
      </div>
    </div>

    <!-- Steps list -->
    <div v-if="chat.taskState?.steps?.length" class="space-y-0.5">
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
