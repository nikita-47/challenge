<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useMCPStore } from '@/stores/mcp'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { MCPToolInfo } from '@/lib/types'

const mcp = useMCPStore()

const selectedServer = ref<string>('')

// Auto-select first server when servers load
watch(() => mcp.servers, (servers) => {
  if (servers.length > 0 && !selectedServer.value) {
    selectedServer.value = servers[0].name
  }
}, { immediate: true })

const currentServer = computed(() =>
  mcp.servers.find((s) => s.name === selectedServer.value)
)

const currentTools = computed<MCPToolInfo[]>(() => {
  if (!selectedServer.value) {
    return []
  }
  return mcp.tools.filter((t) => t.server === selectedServer.value)
})

function formatSchema(schema: unknown): string {
  if (schema === null || schema === undefined) {
    return '{}'
  }
  try {
    return JSON.stringify(schema, null, 2)
  } catch {
    return String(schema)
  }
}

async function toggleConnection() {
  if (!currentServer.value) {
    return
  }
  if (currentServer.value.connected) {
    await mcp.disconnect(currentServer.value.name)
  } else {
    await mcp.connect(currentServer.value.name)
  }
}
</script>

<template>
  <div class="flex flex-col h-full">
    <!-- Server selector -->
    <div class="p-3 border-b border-primary/20 space-y-2">
      <div class="flex items-center gap-2">
        <Select v-model="selectedServer">
          <SelectTrigger class="flex-1 h-7 text-xs bg-background border-primary/30">
            <SelectValue placeholder="select server..." />
          </SelectTrigger>
          <SelectContent>
            <SelectItem
              v-for="server in mcp.servers"
              :key="server.name"
              :value="server.name"
            >
              <span class="flex items-center gap-1.5">
                <span
                  class="inline-block w-1.5 h-1.5 rounded-full shrink-0"
                  :class="server.connected ? 'bg-green-500' : 'bg-red-500'"
                />
                {{ server.name }}
              </span>
            </SelectItem>
          </SelectContent>
        </Select>

        <Button
          v-if="currentServer"
          size="sm"
          variant="outline"
          class="h-7 text-xs px-2 shrink-0"
          :class="currentServer.connected
            ? 'border-destructive/30 text-destructive hover:bg-destructive/10'
            : 'border-primary/30 text-primary hover:bg-primary/10'"
          :disabled="mcp.loading"
          @click="toggleConnection"
        >
          {{ mcp.loading ? '...' : currentServer.connected ? 'off' : 'on' }}
        </Button>
      </div>

      <!-- Status line -->
      <div v-if="currentServer" class="flex items-center gap-2 text-xs">
        <span
          class="inline-block w-2 h-2 rounded-full"
          :class="currentServer.connected ? 'bg-green-500' : 'bg-red-500'"
        />
        <span :class="currentServer.connected ? 'text-primary' : 'text-muted-foreground'">
          {{ currentServer.connected ? `${currentServer.toolsCount} tools` : 'offline' }}
        </span>
        <span v-if="currentServer.error" class="text-destructive truncate">
          {{ currentServer.error }}
        </span>
      </div>

      <div v-if="mcp.error" class="text-xs text-destructive border border-destructive/30 px-2 py-1 bg-destructive/5">
        {{ mcp.error }}
      </div>
    </div>

    <!-- Tools list -->
    <ScrollArea class="flex-1">
      <div class="p-2">
        <!-- No server selected -->
        <div
          v-if="!selectedServer"
          class="text-xs text-muted-foreground p-2"
        >
          // select an MCP server
        </div>

        <!-- Empty state -->
        <div
          v-else-if="!mcp.loading && mcp.servers.length === 0"
          class="text-xs text-muted-foreground p-2"
        >
          // no MCP servers configured
        </div>

        <!-- Loading -->
        <div v-else-if="mcp.loading && currentTools.length === 0" class="text-xs text-muted-foreground p-2">
          loading...
        </div>

        <!-- Not connected -->
        <div
          v-else-if="currentServer && !currentServer.connected"
          class="text-xs text-muted-foreground p-2"
        >
          // server offline — press "on" to connect
        </div>

        <!-- Tools -->
        <div v-else class="space-y-0">
          <div
            v-for="tool in currentTools"
            :key="tool.name"
            class="px-2 py-2 border-b border-primary/5 last:border-b-0 space-y-1"
          >
            <div class="text-xs font-mono text-primary">{{ tool.name }}</div>
            <div v-if="tool.description" class="text-xs text-muted-foreground leading-snug">
              {{ tool.description }}
            </div>
            <details v-if="tool.inputSchema" class="group">
              <summary class="text-[10px] text-muted-foreground/60 cursor-pointer hover:text-muted-foreground select-none">
                schema
              </summary>
              <pre class="mt-1 text-[10px] text-muted-foreground/80 bg-background border border-primary/10 p-2 overflow-x-auto whitespace-pre-wrap break-all font-mono leading-tight">{{ formatSchema(tool.inputSchema) }}</pre>
            </details>
          </div>
        </div>
      </div>
    </ScrollArea>
  </div>
</template>
