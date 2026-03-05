<script setup lang="ts">
import { ref } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { Checkbox } from '@/components/ui/checkbox'
import { Separator } from '@/components/ui/separator'

const props = defineProps<{
  taskMode: boolean
  enabledTools: string[]
}>()

const emit = defineEmits<{
  'update:taskMode': [value: boolean]
  'update:enabledTools': [value: string[]]
}>()

const isOpen = ref(false)
const popoverRef = ref<HTMLDivElement | null>(null)

onClickOutside(popoverRef, () => {
  isOpen.value = false
})

function togglePopover() {
  isOpen.value = !isOpen.value
}

function setTaskMode(value: boolean) {
  emit('update:taskMode', value)
}

function toggleTool(tool: string, checked: boolean) {
  if (checked) {
    if (!props.enabledTools.includes(tool)) {
      emit('update:enabledTools', [...props.enabledTools, tool])
    }
  } else {
    emit('update:enabledTools', props.enabledTools.filter((t) => t !== tool))
  }
}
</script>

<template>
  <div ref="popoverRef" class="relative">
    <button
      class="px-2 py-1 text-xs border transition-colors"
      :class="taskMode
        ? 'bg-cyan-500/20 text-cyan-400 border-cyan-500/40'
        : 'text-muted-foreground border-transparent hover:text-foreground hover:border-border'"
      title="Send settings"
      @click="togglePopover"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="14"
        height="14"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <circle cx="12" cy="12" r="3" />
        <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
      </svg>
    </button>

    <div
      v-if="isOpen"
      class="absolute bottom-full right-0 mb-2 w-48 bg-card border border-border shadow-lg rounded-sm p-3 flex flex-col gap-2 z-50"
    >
      <p class="text-xs text-muted-foreground font-medium">Send settings</p>

      <div class="flex items-center gap-2">
        <Checkbox
          id="task-mode-checkbox"
          :checked="taskMode"
          @update:checked="setTaskMode"
        />
        <label for="task-mode-checkbox" class="text-xs text-foreground cursor-pointer select-none">
          Task mode
        </label>
      </div>

      <template v-if="taskMode">
        <Separator />
        <p class="text-xs text-muted-foreground">Tools</p>
        <div class="flex items-center gap-2">
          <Checkbox
            id="tool-run-shell"
            :checked="enabledTools.includes('run_shell')"
            @update:checked="(v) => toggleTool('run_shell', v)"
          />
          <label for="tool-run-shell" class="text-xs text-foreground cursor-pointer select-none font-mono">
            run_shell
          </label>
        </div>
        <div class="flex items-center gap-2">
          <Checkbox
            id="tool-read-file"
            :checked="enabledTools.includes('read_file')"
            @update:checked="(v) => toggleTool('read_file', v)"
          />
          <label for="tool-read-file" class="text-xs text-foreground cursor-pointer select-none font-mono">
            read_file
          </label>
        </div>
      </template>
    </div>
  </div>
</template>
