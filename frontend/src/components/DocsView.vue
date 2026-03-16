<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useDocsStore } from '@/stores/docs'
import { useUIStore } from '@/stores/ui'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import type { IndexStatus } from '@/lib/types'

const docs = useDocsStore()
const ui = useUIStore()

const fileInput = ref<HTMLInputElement | null>(null)
const selectedFile = ref<File | null>(null)
const chunkSize = ref(1000)
const overlapSize = ref(200)
const expandedChunks = ref<Set<string>>(new Set())

function toggleChunk(chunkId: string) {
  const s = expandedChunks.value
  if (s.has(chunkId)) {
    s.delete(chunkId)
  } else {
    s.add(chunkId)
  }
  expandedChunks.value = new Set(s)
}

function statusBadgeVariant(status: IndexStatus): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (status) {
    case 'ready': {
      return 'default'
    }
    case 'error': {
      return 'destructive'
    }
    default: {
      return 'secondary'
    }
  }
}

function statusBadgeClass(status: IndexStatus): string {
  switch (status) {
    case 'ready': {
      return 'text-emerald-400 border-emerald-400/30 bg-emerald-400/10'
    }
    case 'indexing': {
      return 'text-amber-400 border-amber-400/30 bg-amber-400/10'
    }
    default: {
      return ''
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

function similarityBarClass(sim: number | null): string {
  if (sim === null) {
    return ''
  }
  if (sim > 0.8) {
    return 'bg-emerald-400/70'
  }
  if (sim >= 0.5) {
    return 'bg-amber-400/70'
  }
  return 'bg-red-400/70'
}

function similarityLabel(sim: number): string {
  if (sim > 0.8) {
    return 'Высокая — контент плавно переходит'
  }
  if (sim >= 0.5) {
    return 'Средняя — умеренная смена темы'
  }
  return 'Низкая — резкая смена темы'
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
  await docs.upload(selectedFile.value, chunkSize.value, overlapSize.value)
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
  <TooltipProvider :delay-duration="200">
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

          <Tooltip>
            <TooltipTrigger as-child>
              <div class="flex items-center gap-1 border border-primary/10 px-1.5 py-0.5 rounded-sm">
                <span class="text-[9px] text-muted-foreground/50">size-based:</span>
                <Input
                  v-model="chunkSize"
                  type="number"
                  min="100"
                  max="5000"
                  step="100"
                  class="w-16 !h-6 text-[10px] font-mono !rounded-sm !px-1 !py-0 !shadow-none appearance-none [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
                />
                <span class="text-[9px] text-muted-foreground/30">chars</span>
                <span class="text-[9px] text-muted-foreground/30 mx-0.5">/</span>
                <Input
                  v-model="overlapSize"
                  type="number"
                  min="0"
                  max="2500"
                  step="50"
                  class="w-16 !h-6 text-[10px] font-mono !rounded-sm !px-1 !py-0 !shadow-none appearance-none [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
                />
                <span class="text-[9px] text-muted-foreground/30">overlap</span>
              </div>
            </TooltipTrigger>
            <TooltipContent side="bottom" class="max-w-xs text-xs">
              Параметры для size-based стратегии. Размер чанка (100–5000 символов) и перекрытие между соседними чанками (0 – половина размера).
            </TooltipContent>
          </Tooltip>

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
                    <Badge
                      :variant="statusBadgeVariant(doc.index_status)"
                      class="text-[10px] px-1 py-0"
                      :class="statusBadgeClass(doc.index_status)"
                    >{{ doc.index_status }}</Badge>
                    <span
                      v-if="doc.index_status === 'ready'"
                      class="text-[10px] text-muted-foreground/60"
                    >{{ doc.chunk_count }} chunks</span>
                  </div>
                </div>
                <Button
                  v-if="doc.index_status !== 'indexing'"
                  variant="ghost"
                  size="sm"
                  class="!h-5 !w-5 !p-0 text-[10px] text-muted-foreground/30 hover:text-red-400 opacity-0 group-hover:opacity-100 ml-1 shrink-0 mt-0.5"
                  @click.stop="docs.removeDoc(doc.id)"
                >
                  ×
                </Button>
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

              <Tooltip>
                <TooltipTrigger as-child>
                  <span class="cursor-help">model: <span class="text-primary/70">{{ docs.activeDoc.embedding_model }}</span></span>
                </TooltipTrigger>
                <TooltipContent side="bottom" class="max-w-xs text-xs">
                  Модель для генерации эмбеддингов (векторных представлений текста)
                </TooltipContent>
              </Tooltip>

              <Tooltip>
                <TooltipTrigger as-child>
                  <span class="cursor-help">
                    size-based: <span class="text-primary/70">{{ docs.activeDoc.chunk_size_param }}</span> chars / <span class="text-primary/70">{{ docs.activeDoc.overlap_param }}</span> overlap
                  </span>
                </TooltipTrigger>
                <TooltipContent side="bottom" class="max-w-xs text-xs">
                  Параметры стратегии size-based: размер чанка / перекрытие (в символах)
                </TooltipContent>
              </Tooltip>

              <Badge
                :variant="statusBadgeVariant(docs.activeDoc.index_status)"
                class="text-[10px] px-1 py-0"
                :class="statusBadgeClass(docs.activeDoc.index_status)"
              >{{ docs.activeDoc.index_status }}</Badge>
              <span
                v-if="docs.activeDoc.index_status === 'indexing'"
                class="text-amber-400/50 animate-pulse"
              >indexing...</span>
            </div>

            <!-- Main content -->
            <ScrollArea class="flex-1">
              <div class="p-4 space-y-6 max-w-[calc(100vw-theme(spacing.72)-2rem)]">
                <!-- Chunk comparison section -->
                <div>
                  <div class="text-[10px] text-primary/50 uppercase tracking-wider mb-1">
                    // strategy comparison
                  </div>

                  <Tooltip>
                    <TooltipTrigger as-child>
                      <div class="text-[10px] text-muted-foreground/40 font-mono mb-3 cursor-help w-fit">
                        how this document splits under each approach
                      </div>
                    </TooltipTrigger>
                    <TooltipContent side="bottom" class="max-w-sm text-xs">
                      Документ разбит на чанки 4 стратегиями. Числа — cosine similarity между соседними чанками (0–1). Чем ближе к 1, тем более похож контент. Цвета: зелёный > 0.8, жёлтый 0.5–0.8, красный < 0.5.
                    </TooltipContent>
                  </Tooltip>

                  <!-- Four columns -->
                  <div class="flex gap-4 min-h-0 overflow-x-auto pb-2">
                    <!-- Size column -->
                    <div class="shrink-0 w-64 overflow-hidden border border-primary/10">
                      <div class="px-3 py-2 border-b border-primary/10 bg-primary/3">
                        <Tooltip>
                          <TooltipTrigger as-child>
                            <div class="text-[10px] text-primary/70 font-semibold uppercase tracking-wide cursor-help">
                              size-based ({{ docs.chunkIndex?.size?.chunks?.length ?? 0 }} chunks)
                            </div>
                          </TooltipTrigger>
                          <TooltipContent side="bottom" class="max-w-xs text-xs">
                            Текст нарезается на фрагменты фиксированной длины (chunk_size символов) с перекрытием (overlap). Простой и предсказуемый, но может разрывать предложения и абзацы.
                          </TooltipContent>
                        </Tooltip>

                        <Tooltip v-if="docs.chunkIndex?.size">
                          <TooltipTrigger as-child>
                            <div class="text-[9px] text-muted-foreground/50 font-mono mt-0.5 cursor-help">
                              avg: {{ docs.chunkIndex.size.avg_similarity.toFixed(2) }} | min: {{ docs.chunkIndex.size.min_similarity.toFixed(2) }} | max: {{ docs.chunkIndex.size.max_similarity.toFixed(2) }}
                            </div>
                          </TooltipTrigger>
                          <TooltipContent side="bottom" class="max-w-xs text-xs">
                            Cosine similarity между соседними чанками. avg — средняя похожесть, min — самый резкий переход темы, max — самые похожие соседи.
                          </TooltipContent>
                        </Tooltip>
                      </div>
                      <div class="max-h-96 overflow-y-auto">
                        <div
                          v-if="!docs.chunkIndex || !docs.chunkIndex.size?.chunks?.length"
                          class="text-[10px] text-muted-foreground/40 p-3"
                        >
                          // no chunks
                        </div>
                        <div
                          v-for="chunk in docs.chunkIndex?.size?.chunks"
                          :key="chunk.id"
                          class="relative border-b border-primary/5 px-3 py-2 hover:bg-primary/3 transition-colors"
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
                            <Tooltip v-if="chunk.similarity_to_next !== null">
                              <TooltipTrigger as-child>
                                <span
                                  class="ml-auto text-[9px] font-mono shrink-0 cursor-help"
                                  :class="chunk.similarity_to_next > 0.8 ? 'text-emerald-400/70' : chunk.similarity_to_next >= 0.5 ? 'text-amber-400/70' : 'text-red-400/70'"
                                >
                                  {{ chunk.similarity_to_next.toFixed(2) }}
                                </span>
                              </TooltipTrigger>
                              <TooltipContent side="left" class="max-w-xs text-xs">
                                Cosine similarity: {{ chunk.similarity_to_next.toFixed(4) }}. {{ similarityLabel(chunk.similarity_to_next) }}
                              </TooltipContent>
                            </Tooltip>
                          </div>
                          <div
                            class="text-[10px] text-muted-foreground/80 font-mono leading-relaxed break-all cursor-pointer"
                            :class="expandedChunks.has('size-' + chunk.id) ? '' : 'line-clamp-3'"
                            @click="toggleChunk('size-' + chunk.id)"
                          >
                            {{ expandedChunks.has('size-' + chunk.id) ? chunk.text : (chunk.text.slice(0, 200) + (chunk.text.length > 200 ? '...' : '')) }}
                          </div>
                          <div
                            v-if="chunk.similarity_to_next !== null"
                            class="absolute bottom-0 left-0 h-[2px] w-full"
                            :class="similarityBarClass(chunk.similarity_to_next)"
                          />
                        </div>
                      </div>
                    </div>

                    <!-- Sentence column -->
                    <div class="shrink-0 w-64 overflow-hidden border border-primary/10">
                      <div class="px-3 py-2 border-b border-primary/10 bg-primary/3">
                        <Tooltip>
                          <TooltipTrigger as-child>
                            <div class="text-[10px] text-primary/70 font-semibold uppercase tracking-wide cursor-help">
                              sentence-based ({{ docs.chunkIndex?.sentence?.chunks?.length ?? 0 }} chunks)
                            </div>
                          </TooltipTrigger>
                          <TooltipContent side="bottom" class="max-w-xs text-xs">
                            Текст разбивается по границам предложений. Группирует N предложений в один чанк. Сохраняет целостность предложений, но не учитывает структуру документа.
                          </TooltipContent>
                        </Tooltip>

                        <Tooltip v-if="docs.chunkIndex?.sentence">
                          <TooltipTrigger as-child>
                            <div class="text-[9px] text-muted-foreground/50 font-mono mt-0.5 cursor-help">
                              avg: {{ docs.chunkIndex.sentence.avg_similarity.toFixed(2) }} | min: {{ docs.chunkIndex.sentence.min_similarity.toFixed(2) }} | max: {{ docs.chunkIndex.sentence.max_similarity.toFixed(2) }}
                            </div>
                          </TooltipTrigger>
                          <TooltipContent side="bottom" class="max-w-xs text-xs">
                            Cosine similarity между соседними чанками. avg — средняя похожесть, min — самый резкий переход темы, max — самые похожие соседи.
                          </TooltipContent>
                        </Tooltip>
                      </div>
                      <div class="max-h-96 overflow-y-auto">
                        <div
                          v-if="!docs.chunkIndex || !docs.chunkIndex.sentence?.chunks?.length"
                          class="text-[10px] text-muted-foreground/40 p-3"
                        >
                          // no chunks
                        </div>
                        <div
                          v-for="chunk in docs.chunkIndex?.sentence?.chunks"
                          :key="chunk.id"
                          class="relative border-b border-primary/5 px-3 py-2 hover:bg-primary/3 transition-colors"
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
                            <Tooltip v-if="chunk.similarity_to_next !== null">
                              <TooltipTrigger as-child>
                                <span
                                  class="ml-auto text-[9px] font-mono shrink-0 cursor-help"
                                  :class="chunk.similarity_to_next > 0.8 ? 'text-emerald-400/70' : chunk.similarity_to_next >= 0.5 ? 'text-amber-400/70' : 'text-red-400/70'"
                                >
                                  {{ chunk.similarity_to_next.toFixed(2) }}
                                </span>
                              </TooltipTrigger>
                              <TooltipContent side="left" class="max-w-xs text-xs">
                                Cosine similarity: {{ chunk.similarity_to_next.toFixed(4) }}. {{ similarityLabel(chunk.similarity_to_next) }}
                              </TooltipContent>
                            </Tooltip>
                          </div>
                          <div
                            class="text-[10px] text-muted-foreground/80 font-mono leading-relaxed break-all cursor-pointer"
                            :class="expandedChunks.has('sentence-' + chunk.id) ? '' : 'line-clamp-3'"
                            @click="toggleChunk('sentence-' + chunk.id)"
                          >
                            {{ expandedChunks.has('sentence-' + chunk.id) ? chunk.text : (chunk.text.slice(0, 200) + (chunk.text.length > 200 ? '...' : '')) }}
                          </div>
                          <div
                            v-if="chunk.similarity_to_next !== null"
                            class="absolute bottom-0 left-0 h-[2px] w-full"
                            :class="similarityBarClass(chunk.similarity_to_next)"
                          />
                        </div>
                      </div>
                    </div>

                    <!-- Structure column -->
                    <div class="shrink-0 w-64 overflow-hidden border border-primary/10">
                      <div class="px-3 py-2 border-b border-primary/10 bg-primary/3">
                        <Tooltip>
                          <TooltipTrigger as-child>
                            <div class="text-[10px] text-primary/70 font-semibold uppercase tracking-wide cursor-help">
                              structure-based ({{ docs.chunkIndex?.structure?.chunks?.length ?? 0 }} chunks)
                            </div>
                          </TooltipTrigger>
                          <TooltipContent side="bottom" class="max-w-xs text-xs">
                            Разбивает по структурным элементам документа — заголовки, абзацы, списки, блоки кода. Лучше всего для Markdown/текстов с чёткой структурой.
                          </TooltipContent>
                        </Tooltip>

                        <Tooltip v-if="docs.chunkIndex?.structure">
                          <TooltipTrigger as-child>
                            <div class="text-[9px] text-muted-foreground/50 font-mono mt-0.5 cursor-help">
                              avg: {{ docs.chunkIndex.structure.avg_similarity.toFixed(2) }} | min: {{ docs.chunkIndex.structure.min_similarity.toFixed(2) }} | max: {{ docs.chunkIndex.structure.max_similarity.toFixed(2) }}
                            </div>
                          </TooltipTrigger>
                          <TooltipContent side="bottom" class="max-w-xs text-xs">
                            Cosine similarity между соседними чанками. avg — средняя похожесть, min — самый резкий переход темы, max — самые похожие соседи.
                          </TooltipContent>
                        </Tooltip>
                      </div>
                      <div class="max-h-96 overflow-y-auto">
                        <div
                          v-if="!docs.chunkIndex || !docs.chunkIndex.structure?.chunks?.length"
                          class="text-[10px] text-muted-foreground/40 p-3"
                        >
                          // no chunks
                        </div>
                        <div
                          v-for="chunk in docs.chunkIndex?.structure?.chunks"
                          :key="chunk.id"
                          class="relative border-b border-primary/5 px-3 py-2 hover:bg-primary/3 transition-colors"
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
                            <Tooltip v-if="chunk.similarity_to_next !== null">
                              <TooltipTrigger as-child>
                                <span
                                  class="ml-auto text-[9px] font-mono shrink-0 cursor-help"
                                  :class="chunk.similarity_to_next > 0.8 ? 'text-emerald-400/70' : chunk.similarity_to_next >= 0.5 ? 'text-amber-400/70' : 'text-red-400/70'"
                                >
                                  {{ chunk.similarity_to_next.toFixed(2) }}
                                </span>
                              </TooltipTrigger>
                              <TooltipContent side="left" class="max-w-xs text-xs">
                                Cosine similarity: {{ chunk.similarity_to_next.toFixed(4) }}. {{ similarityLabel(chunk.similarity_to_next) }}
                              </TooltipContent>
                            </Tooltip>
                          </div>
                          <div
                            class="text-[10px] text-muted-foreground/80 font-mono leading-relaxed break-all cursor-pointer"
                            :class="expandedChunks.has('structure-' + chunk.id) ? '' : 'line-clamp-3'"
                            @click="toggleChunk('structure-' + chunk.id)"
                          >
                            {{ expandedChunks.has('structure-' + chunk.id) ? chunk.text : (chunk.text.slice(0, 200) + (chunk.text.length > 200 ? '...' : '')) }}
                          </div>
                          <div
                            v-if="chunk.similarity_to_next !== null"
                            class="absolute bottom-0 left-0 h-[2px] w-full"
                            :class="similarityBarClass(chunk.similarity_to_next)"
                          />
                        </div>
                      </div>
                    </div>

                    <!-- Semantic column -->
                    <div class="shrink-0 w-64 overflow-hidden border border-primary/10">
                      <div class="px-3 py-2 border-b border-primary/10 bg-primary/3">
                        <Tooltip>
                          <TooltipTrigger as-child>
                            <div class="text-[10px] text-primary/70 font-semibold uppercase tracking-wide cursor-help">
                              semantic ({{ docs.chunkIndex?.semantic?.chunks?.length ?? 0 }} chunks)
                            </div>
                          </TooltipTrigger>
                          <TooltipContent side="bottom" class="max-w-xs text-xs">
                            Разбивает текст по смысловым границам, используя эмбеддинги. Сравнивает косинусное сходство между соседними предложениями и режет там, где смысл резко меняется.
                          </TooltipContent>
                        </Tooltip>

                        <Tooltip v-if="docs.chunkIndex?.semantic">
                          <TooltipTrigger as-child>
                            <div class="text-[9px] text-muted-foreground/50 font-mono mt-0.5 cursor-help">
                              avg: {{ docs.chunkIndex.semantic.avg_similarity.toFixed(2) }} | min: {{ docs.chunkIndex.semantic.min_similarity.toFixed(2) }} | max: {{ docs.chunkIndex.semantic.max_similarity.toFixed(2) }}
                            </div>
                          </TooltipTrigger>
                          <TooltipContent side="bottom" class="max-w-xs text-xs">
                            Cosine similarity между соседними чанками. avg — средняя похожесть, min — самый резкий переход темы, max — самые похожие соседи.
                          </TooltipContent>
                        </Tooltip>
                      </div>
                      <div class="max-h-96 overflow-y-auto">
                        <div
                          v-if="!docs.chunkIndex || !docs.chunkIndex.semantic?.chunks?.length"
                          class="text-[10px] text-muted-foreground/40 p-3"
                        >
                          // no chunks
                        </div>
                        <div
                          v-for="chunk in docs.chunkIndex?.semantic?.chunks"
                          :key="chunk.id"
                          class="relative border-b border-primary/5 px-3 py-2 hover:bg-primary/3 transition-colors"
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
                            <Tooltip v-if="chunk.similarity_to_next !== null">
                              <TooltipTrigger as-child>
                                <span
                                  class="ml-auto text-[9px] font-mono shrink-0 cursor-help"
                                  :class="chunk.similarity_to_next > 0.8 ? 'text-emerald-400/70' : chunk.similarity_to_next >= 0.5 ? 'text-amber-400/70' : 'text-red-400/70'"
                                >
                                  {{ chunk.similarity_to_next.toFixed(2) }}
                                </span>
                              </TooltipTrigger>
                              <TooltipContent side="left" class="max-w-xs text-xs">
                                Cosine similarity: {{ chunk.similarity_to_next.toFixed(4) }}. {{ similarityLabel(chunk.similarity_to_next) }}
                              </TooltipContent>
                            </Tooltip>
                          </div>
                          <div
                            class="text-[10px] text-muted-foreground/80 font-mono leading-relaxed break-all cursor-pointer"
                            :class="expandedChunks.has('semantic-' + chunk.id) ? '' : 'line-clamp-3'"
                            @click="toggleChunk('semantic-' + chunk.id)"
                          >
                            {{ expandedChunks.has('semantic-' + chunk.id) ? chunk.text : (chunk.text.slice(0, 200) + (chunk.text.length > 200 ? '...' : '')) }}
                          </div>
                          <div
                            v-if="chunk.similarity_to_next !== null"
                            class="absolute bottom-0 left-0 h-[2px] w-full"
                            :class="similarityBarClass(chunk.similarity_to_next)"
                          />
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
  </TooltipProvider>
</template>
