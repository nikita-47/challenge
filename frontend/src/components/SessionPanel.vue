<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { useSessionsStore } from '@/stores/sessions'
import { useChatStore } from '@/stores/chat'
import { useUIStore } from '@/stores/ui'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'

defineEmits<{ close: [] }>()

const sessions = useSessionsStore()
const chat = useChatStore()
const ui = useUIStore()

const editingName = ref<string | null>(null)
const editValue = ref('')
let renaming = false

function startEdit(name: string) {
  editingName.value = name
  editValue.value = name
  nextTick(() => {
    const el = document.querySelector<HTMLInputElement>('[data-rename-input]')
    el?.focus()
    el?.select()
  })
}

async function confirmRename(oldName: string) {
  if (renaming) {
    return
  }
  const newName = editValue.value.trim()
  editingName.value = null
  if (!newName || newName === oldName) {
    return
  }
  renaming = true
  try {
    await sessions.renameSession(oldName, newName)
  } catch (e) {
    console.error('Rename failed:', e)
  } finally {
    renaming = false
  }
}

function cancelEdit() {
  editingName.value = null
}
</script>

<template>
  <aside class="flex flex-col border-r border-primary/20 bg-card h-full">
    <div class="p-3 border-b border-primary/20 flex items-center justify-between">
      <h2 class="text-xs font-semibold text-primary uppercase tracking-wider">// sessions</h2>
      <Button
        variant="ghost"
        size="icon"
        class="h-6 w-6 text-muted-foreground"
        @click="$emit('close')"
      >
        &times;
      </Button>
    </div>
    <div class="p-3 border-b border-primary/20">
      <Button
        class="w-full"
        size="sm"
        @click="ui.newChatDialogOpen = true"
      >
        + new_chat
      </Button>
    </div>
    <ScrollArea class="flex-1">
      <div class="p-1 space-y-0">
        <div v-if="sessions.loading" class="text-xs text-muted-foreground p-2">
          loading...
        </div>
        <div
          v-for="name in sessions.sessions"
          :key="name"
          class="group flex items-center justify-between px-2 py-1.5 text-sm cursor-pointer transition-colors border-l-2"
          :class="chat.currentSession === name ? 'border-primary text-primary bg-primary/5' : 'border-transparent text-muted-foreground hover:bg-primary/5 hover:text-foreground'"
          @click="editingName !== name && sessions.loadSession(name)"
        >
          <input
            v-if="editingName === name"
            data-rename-input
            v-model="editValue"
            class="bg-background border border-primary/30 px-1 text-sm w-full outline-none text-foreground"
            @keydown.enter="confirmRename(name)"
            @keydown.escape.stop="cancelEdit()"
            @blur="confirmRename(name)"
            @click.stop
          />
          <span
            v-else
            class="truncate"
            @dblclick.stop="startEdit(name)"
          >
            {{ name }}
          </span>
          <Button
            v-if="editingName !== name"
            variant="ghost"
            size="icon"
            class="opacity-0 group-hover:opacity-100 h-6 w-6 shrink-0 text-destructive hover:text-destructive"
            @click.stop="sessions.deleteSession(name)"
            title="Delete session"
          >
            &times;
          </Button>
        </div>
        <div v-if="!sessions.loading && sessions.sessions.length === 0" class="text-xs text-muted-foreground p-2">
          // no saved sessions
        </div>
      </div>
    </ScrollArea>
  </aside>
</template>
