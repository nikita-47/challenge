<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useDocsStore } from '@/stores/docs'
import { useUIStore } from '@/stores/ui'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import type { IndexStatus } from '@/lib/types'

const docs = useDocsStore()
const ui = useUIStore()

const fileInput = ref<HTMLInputElement | null>(null)
const selectedFile = ref<File | null>(null)

function statusBadgeClass(status: IndexStatus): string {
  switch (status) {
    case 'ready': {
      return 'text-emerald-400 border-emerald-400/30 bg-emerald-400/10'
    }
    case 'indexing': {
      return 'text-amber-400 border-amber-400/30 bg-amber-400/10'
    }
    case 'error': {
      return 'text-red-400 border-red-400/30 bg-red-400/10'
    }
    default: {
      return 'text-zinc-400 border-zinc-400/30 bg-zinc-400/10'
    }
  }
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) {
    return `${bytes}B`
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)}KB`
  }
  return `${(bytes / (1024 * 1024)).toFixed(1)}MB`
}

function formatDate(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

function handleFileSelect(e: Event) {
  const target = e.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    selectedFile.value = target.files[0]
  }
}

async function handleUpload() {
  if (!selectedFile.value || docs.uploading) {
    return
  }
  await docs.upload(selectedFile.value)
  selectedFile.value = null
  if (fileInput.value) {
    fileInput.value.value = ''
  }
}

onMounted(() => {
  docs.loadList()
})
</script>

<template>
  <div class="flex flex-col h-screen w-screen bg-background font-mono overflow-hidden">
    <!-- Header bar -->
    <div class="flex items-center gap-4 px-4 py-2 border-b border-primary/20 bg-card shrink-0 flex-wrap">
      <div class="text-sm text-primary font-semibold tracking-widest shrink-0">
        // documents
      </div>

      <!-- Upload controls -->
      <div class="flex flex-1 items-center gap-2 flex-wrap">
        <input
          ref="fileInput"
          type="file"
          accept=".txt,.md,.pdf"
          class="hidden"
          @change="handleFileSelect"
        />
        <Button
          size="sm"
          variant="outline"
          class="h-7 text-xs shrink-0"
          :disabled="docs.uploading"
          @click="fileInput?.click()"
        >
          {{ selectedFile ? selectedFile.name : 'choose file' }}
        </Button>

        <Button
          size="sm"
          class="h-7 px-4 text-xs shrink-0"
          :disabled="docs.uploading || !selectedFile"
          @click="handleUpload"
        >
          {{ docs.uploading ? 'uploading...' : 'upload' }}
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
      v-if="docs.error"
      class="shrink-0 text-xs text-red-400 border-b border-red-400/30 px-4 py-1.5 bg-red-400/5 font-mono"
    >
      {{ docs.error }}
    </div>

    <!-- Body -->
    <div class="flex flex-1 min-h-0">
      <!-- Left column: documents list -->
      <div class="w-72 shrink-0 border-r border-primary/20 flex flex-col">
        <div class="flex items-center justify-between px-3 py-2 border-b border-primary/10">
          <span class="text-[10px] text-primary/60 uppercase tracking-wider">
            // documents ({{ docs.documents.length }})
          </span>
          <button
            class="text-primary/50 hover:text-primary transition-colors text-sm"
            title="Refresh"
            @click="docs.loadList()"
          >
            ↺
          </button>
        </div>

        <ScrollArea class="flex-1">
          <div class="p-2 space-y-1">
            <div
              v-if="docs.documents.length === 0"
              class="text-xs text-muted-foreground/50 font-mono py-3 px-1"
            >
              // no documents yet
            </div>
            <div
              v-for="doc in docs.documents"
              :key="doc.id"
              class="group flex items-start justify-between gap-2 px-2 py-2 border cursor-pointer transition-colors"
              :class="docs.activeDoc?.id === doc.id
                ? 'border-primary/40 bg-primary/5 text-primary'
                : 'border-transparent hover:border-primary/20 hover:bg-primary/5'"
              @click="docs.selectDoc(doc.id)"
            >
              <div class="min-w-0 flex-1 space-y-1">
                <div class="text-xs font-mono text-foreground truncate">
                  {{ doc.original_name }}
                </div>
                <div class="flex items-center gap-2 flex-wrap">
                  <span
                    class="text-[10px] font-mono px-1 border"
                    :class="statusBadgeClass(doc.index_status)"
                  >{{ doc.index_status }}</span>
                  <span
                    v-if="doc.index_status === 'ready'"
                    class="text-[10px] text-muted-foreground/60"
                  >{{ doc.chunk_count }} chunks</span>
                </div>
              </div>
              <button
                v-if="doc.index_status !== 'indexing'"
                class="text-[10px] text-muted-foreground/30 hover:text-red-400 transition-colors opacity-0 group-hover:opacity-100 ml-1 shrink-0 mt-0.5"
                title="delete document"
                @click.stop="docs.removeDoc(doc.id)"
              >
                ×
              </button>
            </div>
          </div>
        </ScrollArea>
      </div>

      <!-- Center area -->
      <div class="flex-1 flex flex-col min-h-0">
        <!-- Empty state -->
        <div
          v-if="!docs.activeDoc"
          class="flex-1 flex items-center justify-center text-muted-foreground/30 text-sm font-mono"
        >
          // select a document
        </div>

        <!-- Document detail -->
        <template v-else>
          <!-- Meta bar -->
          <div class="shrink-0 px-4 py-2.5 border-b border-primary/10 flex items-center gap-4 flex-wrap text-[11px] font-mono text-muted-foreground">
            <span class="text-foreground font-semibold">{{ docs.activeDoc.original_name }}</span>
            <span>{{ formatBytes(docs.activeDoc.size) }}</span>
            <span>{{ docs.activeDoc.content_type }}</span>
            <span>{{ formatDate(docs.activeDoc.uploaded_at) }}</span>
            <span>model: <span class="text-primary/70">{{ docs.activeDoc.embedding_model }}</span></span>
            <span
              v-if="docs.activeDoc.index_status === 'ready'"
            >chunks: <span class="text-emerald-400">{{ docs.activeDoc.chunk_count }}</span></span>
            <span
              class="px-1 border text-[10px]"
              :class="statusBadgeClass(docs.activeDoc.index_status)"
            >{{ docs.activeDoc.index_status }}</span>
            <span
              v-if="docs.activeDoc.index_status === 'indexing'"
              class="text-amber-400/50 animate-pulse"
            >indexing...</span>
          </div>

          <!-- Main content -->
          <ScrollArea class="flex-1">
            <div class="p-4 space-y-6">
              <!-- Chunk comparison section -->
              <div>
                <div class="text-[10px] text-primary/50 uppercase tracking-wider mb-1">
                  // strategy comparison
                </div>
                <div class="text-[10px] text-muted-foreground/40 font-mono mb-3">
                  how this document splits under each approach
                </div>

                <!-- Three columns -->
                <div class="flex gap-4 min-h-0 overflow-hidden">
                  <!-- Size column -->
                  <div class="flex-1 min-w-0 overflow-hidden border border-primary/10">
                    <div class="px-3 py-2 border-b border-primary/10 bg-primary/3">
                      <span class="text-[10px] text-primary/70 font-semibold uppercase tracking-wide">
                        size-based ({{ docs.chunkIndex?.size?.length ?? 0 }} chunks)
                      </span>
                    </div>
                    <div class="max-h-96 overflow-y-auto">
                      <div
                        v-if="!docs.chunkIndex || !docs.chunkIndex.size?.length"
                        class="text-[10px] text-muted-foreground/40 p-3"
                      >
                        // no chunks
                      </div>
                      <div
                        v-for="chunk in docs.chunkIndex?.size"
                        :key="chunk.id"
                        class="border-b border-primary/5 px-3 py-2 hover:bg-primary/3 transition-colors"
                      >
                        <div class="flex items-center gap-1 mb-1 overflow-hidden">
                          <span class="text-[9px] text-primary/50 border border-primary/20 px-1 shrink-0">
                            #{{ chunk.index }}
                          </span>
                          <span
                            v-if="chunk.metadata?.section"
                            class="text-[9px] text-muted-foreground/50 border border-muted/20 px-1 truncate min-w-0"
                          >
                            {{ chunk.metadata.section }}
                          </span>
                        </div>
                        <div class="text-[10px] text-muted-foreground/80 font-mono leading-relaxed line-clamp-3 break-all">
                          {{ chunk.text.slice(0, 200) }}{{ chunk.text.length > 200 ? '…' : '' }}
                        </div>
                      </div>
                    </div>
                  </div>

                  <!-- Sentence column -->
                  <div class="flex-1 min-w-0 overflow-hidden border border-primary/10">
                    <div class="px-3 py-2 border-b border-primary/10 bg-primary/3">
                      <span class="text-[10px] text-primary/70 font-semibold uppercase tracking-wide">
                        sentence-based ({{ docs.chunkIndex?.sentence?.length ?? 0 }} chunks)
                      </span>
                    </div>
                    <div class="max-h-96 overflow-y-auto">
                      <div
                        v-if="!docs.chunkIndex || !docs.chunkIndex.sentence?.length"
                        class="text-[10px] text-muted-foreground/40 p-3"
                      >
                        // no chunks
                      </div>
                      <div
                        v-for="chunk in docs.chunkIndex?.sentence"
                        :key="chunk.id"
                        class="border-b border-primary/5 px-3 py-2 hover:bg-primary/3 transition-colors"
                      >
                        <div class="flex items-center gap-1 mb-1 overflow-hidden">
                          <span class="text-[9px] text-primary/50 border border-primary/20 px-1 shrink-0">
                            #{{ chunk.index }}
                          </span>
                          <span
                            v-if="chunk.metadata?.section"
                            class="text-[9px] text-muted-foreground/50 border border-muted/20 px-1 truncate min-w-0"
                          >
                            {{ chunk.metadata.section }}
                          </span>
                        </div>
                        <div class="text-[10px] text-muted-foreground/80 font-mono leading-relaxed line-clamp-3 break-all">
                          {{ chunk.text.slice(0, 200) }}{{ chunk.text.length > 200 ? '…' : '' }}
                        </div>
                      </div>
                    </div>
                  </div>

                  <!-- Structure column -->
                  <div class="flex-1 min-w-0 overflow-hidden border border-primary/10">
                    <div class="px-3 py-2 border-b border-primary/10 bg-primary/3">
                      <span class="text-[10px] text-primary/70 font-semibold uppercase tracking-wide">
                        structure-based ({{ docs.chunkIndex?.structure?.length ?? 0 }} chunks)
                      </span>
                    </div>
                    <div class="max-h-96 overflow-y-auto">
                      <div
                        v-if="!docs.chunkIndex || !docs.chunkIndex.structure?.length"
                        class="text-[10px] text-muted-foreground/40 p-3"
                      >
                        // no chunks
                      </div>
                      <div
                        v-for="chunk in docs.chunkIndex?.structure"
                        :key="chunk.id"
                        class="border-b border-primary/5 px-3 py-2 hover:bg-primary/3 transition-colors"
                      >
                        <div class="flex items-center gap-1 mb-1 overflow-hidden">
                          <span class="text-[9px] text-primary/50 border border-primary/20 px-1 shrink-0">
                            #{{ chunk.index }}
                          </span>
                          <span
                            v-if="chunk.metadata?.section"
                            class="text-[9px] text-muted-foreground/50 border border-muted/20 px-1 truncate min-w-0"
                          >
                            {{ chunk.metadata.section }}
                          </span>
                        </div>
                        <div class="text-[10px] text-muted-foreground/80 font-mono leading-relaxed line-clamp-3 break-all">
                          {{ chunk.text.slice(0, 200) }}{{ chunk.text.length > 200 ? '…' : '' }}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <!-- Error detail -->
              <div
                v-if="docs.activeDoc.index_status === 'error' && docs.activeDoc.index_error"
                class="border border-red-400/20 bg-red-400/5 px-3 py-2"
              >
                <div class="text-[10px] text-red-400/70 uppercase tracking-wider mb-1">
                  // index error
                </div>
                <div class="text-xs text-red-400 font-mono">
                  {{ docs.activeDoc.index_error }}
                </div>
              </div>
            </div>
          </ScrollArea>
        </template>
      </div>
    </div>
  </div>
</template>
