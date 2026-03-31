import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { PullRequest, ReviewStep } from '@/lib/types'
import { fetchOpenPRs } from '@/lib/api'
import { streamRequest } from '@/composables/useSSE'

export const useReviewStore = defineStore('review', () => {
  const prs = ref<PullRequest[]>([])
  const selectedPR = ref<PullRequest | null>(null)
  const loading = ref(false)
  const reviewing = ref(false)
  const reviewText = ref('')
  const reviewSteps = ref<ReviewStep[]>([])
  const error = ref<string | null>(null)
  let abortController: AbortController | null = null

  async function loadPRs() {
    loading.value = true
    error.value = null
    try {
      prs.value = await fetchOpenPRs()
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      loading.value = false
    }
  }

  function selectPR(pr: PullRequest) {
    selectedPR.value = pr
    reviewText.value = ''
    reviewSteps.value = []
    error.value = null
  }

  async function runReview() {
    if (!selectedPR.value || reviewing.value) {
      return
    }
    reviewing.value = true
    reviewText.value = ''
    reviewSteps.value = []
    error.value = null
    abortController = new AbortController()

    try {
      await streamRequest(
        '/api/review/run',
        { pr_number: selectedPR.value.number },
        (event) => {
          switch (event.type) {
            case 'review_step': {
              const stepEvent = event as { type: 'review_step'; step: string; status: string; detail?: string }
              const existing = reviewSteps.value.findIndex((s) => s.step === stepEvent.step)
              const step: ReviewStep = {
                step: stepEvent.step as ReviewStep['step'],
                status: stepEvent.status as ReviewStep['status'],
                detail: stepEvent.detail,
              }
              if (existing >= 0) {
                reviewSteps.value[existing] = step
              } else {
                reviewSteps.value.push(step)
              }
              break
            }
            case 'text_delta': {
              reviewText.value += (event as { type: 'text_delta'; text: string }).text
              break
            }
            case 'error': {
              error.value = (event as { type: 'error'; message?: string }).message ?? 'Review failed'
              break
            }
            case 'done': {
              break
            }
          }
        },
        abortController.signal,
      )
    } catch (e) {
      if ((e as Error).name !== 'AbortError') {
        error.value = e instanceof Error ? e.message : String(e)
      }
    } finally {
      reviewing.value = false
      abortController = null
    }
  }

  function cancelReview() {
    abortController?.abort()
  }

  return {
    prs,
    selectedPR,
    loading,
    reviewing,
    reviewText,
    reviewSteps,
    error,
    loadPRs,
    selectPR,
    runReview,
    cancelReview,
  }
})
