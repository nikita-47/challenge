<script setup lang="ts">
import { ref, nextTick, watch } from 'vue'
import { useSessionsStore } from '@/stores/sessions'
import { useMemoryStore } from '@/stores/memory'
import { useChatStore } from '@/stores/chat'
import { useUIStore } from '@/stores/ui'
import { useMCPStore } from '@/stores/mcp'
import { fetchProfile, fetchProject, fetchOperator, updateProfile, updateProject, updateOperator } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import MemoryEditorDialog from '@/components/MemoryEditorDialog.vue'
import GlobalSettings from '@/components/GlobalSettings.vue'
import MCPPanel from '@/components/MCPPanel.vue'

defineEmits<{ close: [] }>()

const sessions = useSessionsStore()
const memory = useMemoryStore()
const chat = useChatStore()
const ui = useUIStore()
const mcpStore = useMCPStore()

const tab = ref<'sessions' | 'memory' | 'mcp'>('sessions')

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

// Memory editor state
const editorOpen = ref(false)
const editorMode = ref<'create' | 'edit'>('create')
const editorKind = ref<'profile' | 'project' | 'operator'>('profile')
const editorName = ref('')
const editorContent = ref('')

function openCreateDialog(kind: 'profile' | 'project' | 'operator') {
  editorKind.value = kind
  editorMode.value = 'create'
  editorName.value = ''
  editorContent.value = ''
  editorOpen.value = true
}

async function openEditDialog(kind: 'profile' | 'project' | 'operator', name: string) {
  editorKind.value = kind
  editorMode.value = 'edit'
  editorName.value = name
  try {
    const fetcher = kind === 'profile' ? fetchProfile : kind === 'project' ? fetchProject : fetchOperator
    const data = await fetcher(name)
    editorContent.value = data.content
  } catch {
    editorContent.value = ''
  }
  editorOpen.value = true
}

async function handleEditorSave(name: string, content: string) {
  if (editorMode.value === 'create') {
    if (editorKind.value === 'profile') {
      await memory.addProfile(name, content)
    } else if (editorKind.value === 'project') {
      await memory.addProject(name, content)
    } else {
      await memory.addOperator(name, content)
    }
  } else {
    if (editorKind.value === 'profile') {
      await updateProfile(name, content)
    } else if (editorKind.value === 'project') {
      await updateProject(name, content)
    } else {
      await updateOperator(name, content)
    }
  }
}

async function handleDelete(kind: 'profile' | 'project' | 'operator', name: string) {
  if (kind === 'profile') {
    await memory.removeProfile(name)
  } else if (kind === 'project') {
    await memory.removeProject(name)
  } else {
    await memory.removeOperator(name)
  }
}

watch(tab, (v) => {
  if (v === 'memory') {
    memory.loadAll()
  }
  if (v === 'mcp') {
    mcpStore.loadServers()
    mcpStore.loadTools()
  }
})
</script>

<template>
  <aside class="flex flex-col border-r border-primary/20 bg-card h-full">
    <div class="p-3 border-b border-primary/20 flex items-center justify-between">
      <div class="flex items-center gap-1">
        <button
          class="px-2 py-0.5 text-xs transition-colors border"
          :class="tab === 'sessions' ? 'bg-primary/10 text-primary border-primary/30' : 'text-muted-foreground hover:text-foreground border-transparent'"
          @click="tab = 'sessions'"
        >
          sessions
        </button>
        <button
          class="px-2 py-0.5 text-xs transition-colors border"
          :class="tab === 'memory' ? 'bg-primary/10 text-primary border-primary/30' : 'text-muted-foreground hover:text-foreground border-transparent'"
          @click="tab = 'memory'"
        >
          memory
        </button>
        <button
          class="px-2 py-0.5 text-xs transition-colors border"
          :class="tab === 'mcp' ? 'bg-primary/10 text-primary border-primary/30' : 'text-muted-foreground hover:text-foreground border-transparent'"
          @click="tab = 'mcp'"
        >
          mcp
        </button>
      </div>
      <Button
        variant="ghost"
        size="icon"
        class="h-6 w-6 text-muted-foreground"
        @click="$emit('close')"
      >
        &times;
      </Button>
    </div>

    <!-- Sessions tab -->
    <template v-if="tab === 'sessions'">
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
            v-for="s in sessions.sessions"
            :key="s.name"
            class="group flex items-center justify-between px-2 py-1.5 text-sm cursor-pointer transition-colors border-l-2"
            :class="chat.currentSession === s.name ? 'border-primary text-primary bg-primary/5' : 'border-transparent text-muted-foreground hover:bg-primary/5 hover:text-foreground'"
            @click="editingName !== s.name && sessions.loadSession(s.name)"
          >
            <input
              v-if="editingName === s.name"
              data-rename-input
              v-model="editValue"
              class="bg-background border border-primary/30 px-1 text-sm w-full outline-none text-foreground"
              @keydown.enter="confirmRename(s.name)"
              @keydown.escape.stop="cancelEdit()"
              @blur="confirmRename(s.name)"
              @click.stop
            />
            <div v-else class="min-w-0 flex-1" @dblclick.stop="startEdit(s.name)">
              <div class="truncate">{{ s.name }}</div>
              <div v-if="s.operator || s.profile || s.project" class="flex gap-1 mt-0.5">
                <span v-if="s.operator" class="text-[10px] text-yellow-400/60 border border-yellow-400/20 px-1 leading-tight">{{ s.operator }}</span>
                <span v-if="s.profile" class="text-[10px] text-primary/60 border border-primary/20 px-1 leading-tight">{{ s.profile }}</span>
                <span v-if="s.project" class="text-[10px] text-accent-foreground/60 border border-accent-foreground/20 px-1 leading-tight">{{ s.project }}</span>
              </div>
            </div>
            <Button
              v-if="editingName !== s.name"
              variant="ghost"
              size="icon"
              class="opacity-0 group-hover:opacity-100 h-6 w-6 shrink-0 text-destructive hover:text-destructive"
              @click.stop="sessions.deleteSession(s.name)"
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
    </template>

    <!-- Memory tab -->
    <ScrollArea v-else-if="tab === 'memory'" class="flex-1">
      <div class="p-3 space-y-3 text-sm">
        <!-- Operators -->
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <div class="text-xs font-medium text-primary uppercase tracking-wider">
              // operators
            </div>
            <button
              class="text-xs text-primary hover:text-accent-foreground"
              @click="openCreateDialog('operator')"
            >+</button>
          </div>
          <div v-if="memory.operators.length === 0" class="text-xs text-muted-foreground">
            No operators yet.
          </div>
          <div
            v-for="p in memory.operators"
            :key="p"
            class="flex items-center justify-between text-xs bg-background border border-border p-1.5 cursor-pointer hover:border-primary/30"
            @click="openEditDialog('operator', p)"
          >
            <span class="text-foreground">{{ p }}</span>
            <button
              class="text-muted-foreground hover:text-destructive ml-2"
              @click.stop="handleDelete('operator', p)"
            >&times;</button>
          </div>
        </div>

        <div class="border-t border-primary/10 my-2" />

        <!-- Profiles -->
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <div class="text-xs font-medium text-primary uppercase tracking-wider">
              // profiles
            </div>
            <button
              class="text-xs text-primary hover:text-accent-foreground"
              @click="openCreateDialog('profile')"
            >+</button>
          </div>
          <div v-if="memory.profiles.length === 0" class="text-xs text-muted-foreground">
            No profiles yet.
          </div>
          <div
            v-for="p in memory.profiles"
            :key="p"
            class="flex items-center justify-between text-xs bg-background border border-border p-1.5 cursor-pointer hover:border-primary/30"
            @click="openEditDialog('profile', p)"
          >
            <span class="text-foreground">{{ p }}</span>
            <button
              class="text-muted-foreground hover:text-destructive ml-2"
              @click.stop="handleDelete('profile', p)"
            >&times;</button>
          </div>
        </div>

        <div class="border-t border-primary/10 my-2" />

        <!-- Projects -->
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <div class="text-xs font-medium text-primary uppercase tracking-wider">
              // projects
            </div>
            <button
              class="text-xs text-primary hover:text-accent-foreground"
              @click="openCreateDialog('project')"
            >+</button>
          </div>
          <div v-if="memory.projects.length === 0" class="text-xs text-muted-foreground">
            No projects yet.
          </div>
          <div
            v-for="p in memory.projects"
            :key="p"
            class="flex items-center justify-between text-xs bg-background border border-border p-1.5 cursor-pointer hover:border-primary/30"
            @click="openEditDialog('project', p)"
          >
            <span class="text-foreground">{{ p }}</span>
            <button
              class="text-muted-foreground hover:text-destructive ml-2"
              @click.stop="handleDelete('project', p)"
            >&times;</button>
          </div>
        </div>
      </div>
    </ScrollArea>

    <!-- MCP tab -->
    <MCPPanel v-else-if="tab === 'mcp'" class="flex-1 min-h-0" />

    <!-- Pipeline view button -->
    <div class="shrink-0 px-3 py-2 border-t border-primary/10">
      <button
        class="w-full text-left text-xs font-mono text-muted-foreground hover:text-primary transition-colors px-2 py-1.5 border border-transparent hover:border-primary/20 hover:bg-primary/5"
        @click="ui.setView('pipeline')"
      >
        ▸ pipeline
      </button>
    </div>

    <GlobalSettings />

    <MemoryEditorDialog
      :open="editorOpen"
      :mode="editorMode"
      :kind="editorKind"
      :initial-name="editorName"
      :initial-content="editorContent"
      @update:open="editorOpen = $event"
      @save="handleEditorSave"
    />
  </aside>
</template>
