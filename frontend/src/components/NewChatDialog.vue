<script setup lang="ts">
import { ref, watch } from 'vue'
import { useUIStore } from '@/stores/ui'
import { useSessionsStore } from '@/stores/sessions'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Slider } from '@/components/ui/slider'
import { Textarea } from '@/components/ui/textarea'
import { Button } from '@/components/ui/button'
import type { ChatSettings, ContextStrategy } from '@/lib/types'

const strategies: { value: ContextStrategy; label: string }[] = [
  { value: 'summary', label: 'Summary (default)' },
  { value: 'window', label: 'Sliding Window' },
  { value: 'facts', label: 'Sticky Facts' },
  { value: 'branch', label: 'Branching' },
]

const models = [
  'claude-sonnet-4-5-20250929',
  'claude-3-5-haiku-20241022',
  'claude-opus-4-20250514',
  'gpt-4o-mini',
  'gpt-4o',
]

const ui = useUIStore()
const sessions = useSessionsStore()

const chatName = ref('')
const model = ref('')
const temperature = ref([0.7])
const maxTokens = ref(1024)
const system = ref('')
const strategy = ref<ContextStrategy>('summary')
const windowSize = ref(10)

watch(() => ui.newChatDialogOpen, (open) => {
  if (open) {
    chatName.value = ''
    model.value = ui.config?.model ?? ''
    const t = ui.config?.temperature
    temperature.value = [t != null && t >= 0 ? t : 0.7]
    maxTokens.value = ui.config?.maxTokens ?? 1024
    system.value = ui.config?.system ?? ''
    strategy.value = 'summary'
    windowSize.value = 10
  }
})

function confirm() {
  const s: ChatSettings = {
    model: model.value,
    temperature: temperature.value[0] ?? 0.7,
    maxTokens: maxTokens.value,
    system: system.value,
    strategy: strategy.value,
    windowSize: strategy.value !== 'branch' ? windowSize.value : undefined,
  }
  sessions.newChat(chatName.value || undefined, s)
  ui.newChatDialogOpen = false
}
</script>

<template>
  <Dialog v-model:open="ui.newChatDialogOpen">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>New Chat</DialogTitle>
        <DialogDescription>Configure settings for the new chat session.</DialogDescription>
      </DialogHeader>

      <div class="space-y-4 py-2">
        <div class="space-y-2">
          <Label for="chat-name">Chat name</Label>
          <Input
            id="chat-name"
            v-model="chatName"
            placeholder="Auto-generated if empty"
          />
        </div>

        <div class="space-y-2">
          <Label>Model</Label>
          <Select v-model="model">
            <SelectTrigger>
              <SelectValue placeholder="Select model" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="m in models" :key="m" :value="m">
                {{ m }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div class="space-y-2">
          <Label>Context Strategy</Label>
          <Select v-model="strategy">
            <SelectTrigger>
              <SelectValue placeholder="Select strategy" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="s in strategies" :key="s.value" :value="s.value">
                {{ s.label }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div v-if="strategy === 'summary' || strategy === 'window' || strategy === 'facts'" class="space-y-2">
          <Label for="window-size">Window size (messages)</Label>
          <Input
            id="window-size"
            type="number"
            v-model="windowSize"
            :min="2"
            :max="100"
          />
        </div>

        <div class="space-y-2">
          <Label>Temperature: {{ (temperature[0] ?? 0.7).toFixed(2) }}</Label>
          <Slider
            v-model="temperature"
            :min="0"
            :max="1"
            :step="0.05"
          />
        </div>

        <div class="space-y-2">
          <Label for="max-tokens">Max tokens</Label>
          <Input
            id="max-tokens"
            type="number"
            v-model="maxTokens"
            :min="1"
            :max="8192"
          />
        </div>

        <div class="space-y-2">
          <Label for="system-prompt">System prompt</Label>
          <Textarea
            id="system-prompt"
            v-model="system"
            rows="3"
            placeholder="Optional system prompt..."
            class="resize-none"
          />
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="ui.newChatDialogOpen = false">Cancel</Button>
        <Button @click="confirm">Create</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
