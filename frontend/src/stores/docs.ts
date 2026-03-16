import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { DocumentMeta, ChunkIndex } from '@/lib/types'
import { fetchDocs, uploadDoc, fetchDoc, deleteDoc, fetchDocChunks } from '@/lib/api'

export const useDocsStore = defineStore('docs', () => {
  const documents = ref<DocumentMeta[]>([])
  const activeDoc = ref<DocumentMeta | null>(null)
  const chunkIndex = ref<ChunkIndex | null>(null)
  const loading = ref(false)
  const uploading = ref(false)
  const error = ref<string | null>(null)
  let pollTimer: ReturnType<typeof setInterval> | null = null

  async function loadList(): Promise<void> {
    loading.value = true
    error.value = null
    try {
      documents.value = await fetchDocs()
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      loading.value = false
    }
  }

  async function upload(file: File): Promise<void> {
    uploading.value = true
    error.value = null
    try {
      const doc = await uploadDoc(file)
      documents.value = [doc, ...documents.value]
      await selectDoc(doc.id)
      startPolling(doc.id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      uploading.value = false
    }
  }

  async function selectDoc(id: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const doc = await fetchDoc(id)
      activeDoc.value = doc
      const idx = documents.value.findIndex((d) => d.id === id)
      if (idx !== -1) {
        documents.value[idx] = doc
      }
      if (doc.index_status === 'ready') {
        chunkIndex.value = await fetchDocChunks(id)
      } else {
        chunkIndex.value = null
      }
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      loading.value = false
    }
  }

  async function removeDoc(id: string): Promise<void> {
    try {
      await deleteDoc(id)
      if (activeDoc.value?.id === id) {
        activeDoc.value = null
        chunkIndex.value = null
        stopPolling()
      }
      documents.value = documents.value.filter((d) => d.id !== id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
    }
  }

  function startPolling(id: string): void {
    stopPolling()
    pollTimer = setInterval(async () => {
      try {
        const doc = await fetchDoc(id)
        activeDoc.value = doc
        const idx = documents.value.findIndex((d) => d.id === id)
        if (idx !== -1) {
          documents.value[idx] = doc
        }
        if (doc.index_status === 'ready' || doc.index_status === 'error') {
          stopPolling()
          if (doc.index_status === 'ready') {
            chunkIndex.value = await fetchDocChunks(id)
          }
        }
      } catch (e) {
        console.error('Polling error:', e)
        stopPolling()
      }
    }, 2000)
  }

  function stopPolling(): void {
    if (pollTimer !== null) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  return {
    documents,
    activeDoc,
    chunkIndex,
    loading,
    uploading,
    error,
    loadList,
    upload,
    selectDoc,
    removeDoc,
    startPolling,
    stopPolling,
  }
})
