<script setup lang="ts">
import { ref, computed } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { Checkbox } from '@/components/ui/checkbox'
import { Separator } from '@/components/ui/separator'
import { useMCPStore } from '@/stores/mcp'
import { useDocsStore } from '@/stores/docs'
import { useChatStore } from '@/stores/chat'

const props = defineProps<{
  taskMode: boolean
  enabledTools: string[]
  invariants: string[]
  mcpTools: string[]
}>()

const emit = defineEmits<{
  'update:taskMode': [value: boolean]
  'update:enabledTools': [value: string[]]
  'update:invariants': [value: string[]]
  'update:mcpTools': [value: string[]]
}>()

const mcp = useMCPStore()
const docs = useDocsStore()
const chat = useChatStore()

const readyDocuments = computed(() => docs.documents.filter((d) => d.index_status === 'ready'))

const connectedServers = computed(() => {
  const grouped: { name: string; tools: typeof mcp.tools.value }[] = []
  for (const srv of mcp.servers) {
    if (!srv.connected) {
      continue
    }
    const serverTools = mcp.tools.filter((t) => t.server === srv.name)
    if (serverTools.length > 0) {
      grouped.push({ name: srv.name, tools: serverTools })
    }
  }
  return grouped
})

const isOpen = ref(false)
const popoverRef = ref<HTMLDivElement | null>(null)
const newInvariant = ref('')

onClickOutside(popoverRef, () => {
  isOpen.value = false
})

function togglePopover() {
  isOpen.value = !isOpen.value
  if (isOpen.value) {
    if (mcp.servers.length === 0) {
      mcp.loadServers()
      mcp.loadTools()
    }
    docs.loadList()
  }
}

function toggleRagDoc(docId: string, checked: boolean) {
  if (checked) {
    if (!chat.activeRagDocIds.includes(docId)) {
      chat.activeRagDocIds.push(docId)
    }
  } else {
    const idx = chat.activeRagDocIds.indexOf(docId)
    if (idx !== -1) {
      chat.activeRagDocIds.splice(idx, 1)
    }
  }
}

function toggleMcpTool(toolId: string, checked: boolean) {
  if (checked) {
    if (!props.mcpTools.includes(toolId)) {
      emit('update:mcpTools', [...props.mcpTools, toolId])
    }
  } else {
    emit('update:mcpTools', props.mcpTools.filter((t) => t !== toolId))
  }
}

function isServerFullySelected(serverName: string) {
  const srv = connectedServers.value.find((s) => s.name === serverName)
  if (!srv) {
    return false
  }
  return srv.tools.every((t) => props.mcpTools.includes(`${serverName}__${t.name}`))
}

function isServerPartiallySelected(serverName: string) {
  const srv = connectedServers.value.find((s) => s.name === serverName)
  if (!srv) {
    return false
  }
  const selected = srv.tools.filter((t) => props.mcpTools.includes(`${serverName}__${t.name}`))
  return selected.length > 0 && selected.length < srv.tools.length
}

function toggleServer(serverName: string, checked: boolean) {
  const srv = connectedServers.value.find((s) => s.name === serverName)
  if (!srv) {
    return
  }
  const serverToolIds = srv.tools.map((t) => `${serverName}__${t.name}`)
  if (checked) {
    const merged = new Set([...props.mcpTools, ...serverToolIds])
    emit('update:mcpTools', [...merged])
  } else {
    emit('update:mcpTools', props.mcpTools.filter((t) => !serverToolIds.includes(t)))
  }
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

function addInvariant() {
  const text = newInvariant.value.trim()
  if (!text) {
    return
  }
  emit('update:invariants', [...props.invariants, text])
  newInvariant.value = ''
}

function removeInvariant(index: number) {
  emit('update:invariants', props.invariants.filter((_, i) => i !== index))
}
</script>

<template>
  <div ref="popoverRef" class="relative">
    <button
      class="px-2 py-1 text-xs border transition-colors"
      :class="taskMode
        ? 'bg-cyan-500/20 text-cyan-400 border-cyan-500/40'
        : mcpTools.length > 0
          ? 'bg-violet-500/20 text-violet-400 border-violet-500/40'
          : chat.activeRagDocIds.length > 0
            ? 'bg-emerald-500/20 text-emerald-400 border-emerald-500/40'
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
      class="absolute bottom-full right-0 mb-2 w-72 max-h-80 overflow-y-auto bg-card border border-border shadow-lg rounded-sm p-3 flex flex-col gap-2 z-50"
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

        <Separator />
        <p class="text-xs text-muted-foreground">Invariants</p>

        <div v-if="invariants.length > 0" class="flex flex-col gap-1">
          <div
            v-for="(inv, index) in invariants"
            :key="index"
            class="flex items-start gap-1.5"
          >
            <span class="text-red-400/70 text-[10px] shrink-0 mt-0.5">!</span>
            <span class="text-xs text-foreground flex-1 break-words">{{ inv }}</span>
            <button
              class="text-muted-foreground hover:text-red-400 text-xs shrink-0 leading-none"
              @click="removeInvariant(index)"
            >
              &times;
            </button>
          </div>
        </div>

        <div class="flex gap-1">
          <input
            v-model="newInvariant"
            type="text"
            placeholder="Add rule..."
            class="flex-1 text-xs px-2 py-1 bg-background border border-border text-foreground placeholder:text-muted-foreground focus:outline-none focus:border-primary/40 min-w-0"
            @keydown.enter.prevent="addInvariant"
          />
          <button
            class="px-2 py-1 text-xs border border-border text-muted-foreground hover:text-foreground hover:border-border/80 shrink-0"
            @click="addInvariant"
          >
            +
          </button>
        </div>
      </template>

      <template v-if="connectedServers.length > 0">
        <Separator />
        <p class="text-xs text-muted-foreground font-medium">MCP Servers</p>
        <div v-for="srv in connectedServers" :key="srv.name" class="flex flex-col gap-1.5">
          <div class="flex items-center gap-2">
            <Checkbox
              :id="`mcp-srv-${srv.name}`"
              :checked="isServerFullySelected(srv.name) || isServerPartiallySelected(srv.name)"
              @update:checked="(v) => toggleServer(srv.name, !isServerFullySelected(srv.name))"
            />
            <label
              :for="`mcp-srv-${srv.name}`"
              class="text-xs text-foreground cursor-pointer select-none font-medium"
            >
              {{ srv.name }}
            </label>
            <span class="text-[10px] text-muted-foreground ml-auto">
              {{ srv.tools.filter(t => mcpTools.includes(`${srv.name}__${t.name}`)).length }}/{{ srv.tools.length }}
            </span>
          </div>
          <div class="pl-5 flex flex-col gap-1">
            <div
              v-for="tool in srv.tools"
              :key="`${srv.name}__${tool.name}`"
              class="flex items-center gap-2"
            >
              <Checkbox
                :id="`mcp-${srv.name}-${tool.name}`"
                :checked="mcpTools.includes(`${srv.name}__${tool.name}`)"
                @update:checked="(v) => toggleMcpTool(`${srv.name}__${tool.name}`, v)"
              />
              <label
                :for="`mcp-${srv.name}-${tool.name}`"
                class="text-xs text-foreground/80 cursor-pointer select-none font-mono"
              >
                {{ tool.name }}
              </label>
            </div>
          </div>
        </div>
      </template>

      <Separator />
      <p class="text-xs text-muted-foreground font-medium">Documents</p>
      <template v-if="readyDocuments.length > 0">
        <div
          v-for="doc in readyDocuments"
          :key="doc.id"
          class="flex items-center gap-2"
        >
          <Checkbox
            :id="`rag-doc-${doc.id}`"
            :checked="chat.activeRagDocIds.includes(doc.id)"
            @update:checked="(v) => toggleRagDoc(doc.id, v)"
          />
          <label
            :for="`rag-doc-${doc.id}`"
            class="text-xs text-foreground/80 cursor-pointer select-none truncate"
            :title="doc.original_name"
          >
            {{ doc.original_name }}
          </label>
        </div>
      </template>
      <p v-else class="text-xs text-muted-foreground/60">
        no indexed documents
      </p>
    </div>
  </div>
</template>
