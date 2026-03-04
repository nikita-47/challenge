import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  fetchProfiles,
  fetchProjects,
  fetchOperators,
  createProfile,
  createProject,
  createOperator,
  deleteProfileAPI,
  deleteProjectAPI,
  deleteOperatorAPI,
} from '@/lib/api'

export const useMemoryStore = defineStore('memory', () => {
  const profiles = ref<string[]>([])
  const projects = ref<string[]>([])
  const operators = ref<string[]>([])

  async function loadProfiles() {
    try {
      profiles.value = await fetchProfiles()
    } catch {
      profiles.value = []
    }
  }

  async function loadProjects() {
    try {
      projects.value = await fetchProjects()
    } catch {
      projects.value = []
    }
  }

  async function loadOperators() {
    try {
      operators.value = await fetchOperators()
    } catch {
      operators.value = []
    }
  }

  async function loadAll() {
    await Promise.all([loadProfiles(), loadProjects(), loadOperators()])
  }

  async function addProfile(name: string, content: string) {
    await createProfile(name, content)
    await loadProfiles()
  }

  async function removeProfile(name: string) {
    await deleteProfileAPI(name)
    profiles.value = profiles.value.filter((p) => p !== name)
  }

  async function addProject(name: string, content: string) {
    await createProject(name, content)
    await loadProjects()
  }

  async function removeProject(name: string) {
    await deleteProjectAPI(name)
    projects.value = projects.value.filter((p) => p !== name)
  }

  async function addOperator(name: string, content: string) {
    await createOperator(name, content)
    await loadOperators()
  }

  async function removeOperator(name: string) {
    await deleteOperatorAPI(name)
    operators.value = operators.value.filter((p) => p !== name)
  }

  return {
    profiles,
    projects,
    operators,
    loadProfiles,
    loadProjects,
    loadOperators,
    loadAll,
    addProfile,
    removeProfile,
    addProject,
    removeProject,
    addOperator,
    removeOperator,
  }
})
