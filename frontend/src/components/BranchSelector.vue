<script setup lang="ts">
import { ref } from 'vue'
import { useChatStore } from '@/stores/chat'
import { useSessionsStore } from '@/stores/sessions'
import { createBranchAPI, switchBranchAPI } from '@/lib/api'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

const chat = useChatStore()
const sessions = useSessionsStore()

const showInput = ref(false)
const newBranchName = ref('')

async function onBranchSwitch(name: string) {
  try {
    await switchBranchAPI(chat.currentSession, name)
    await sessions.loadSession(chat.currentSession)
  } catch (e) {
    console.error('Failed to switch branch:', e)
  }
}

async function createBranch() {
  const name = newBranchName.value.trim()
  if (!name) {
    return
  }
  try {
    await createBranchAPI(chat.currentSession, name)
    await sessions.loadSession(chat.currentSession)
    newBranchName.value = ''
    showInput.value = false
  } catch (e) {
    console.error('Failed to create branch:', e)
  }
}

function cancelCreate() {
  showInput.value = false
  newBranchName.value = ''
}
</script>

<template>
  <div class="flex items-center gap-1">
    <Select :model-value="chat.activeBranch || 'main'" @update:model-value="onBranchSwitch">
      <SelectTrigger class="h-7 w-auto min-w-24 text-xs">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="main">
          main ({{ chat.messages.length }})
        </SelectItem>
        <SelectItem
          v-for="b in chat.branches"
          :key="b.name"
          :value="b.name"
        >
          {{ b.name }} ({{ b.messageCount }})
        </SelectItem>
      </SelectContent>
    </Select>

    <template v-if="showInput">
      <Input
        v-model="newBranchName"
        class="h-7 w-32 text-xs"
        placeholder="Branch name"
        @keyup.enter="createBranch"
        @keyup.escape="cancelCreate"
      />
      <Button variant="ghost" size="sm" class="h-7 px-2 text-xs" @click="createBranch">OK</Button>
      <Button variant="ghost" size="sm" class="h-7 px-2 text-xs" @click="cancelCreate">X</Button>
    </template>
    <Button v-else variant="ghost" size="sm" class="h-7 w-7 p-0 text-xs" @click="showInput = true">+</Button>
  </div>
</template>
