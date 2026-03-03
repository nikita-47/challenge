<script setup lang="ts">
import { ref, watch } from 'vue'
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
import { Textarea } from '@/components/ui/textarea'
import { Button } from '@/components/ui/button'

const props = defineProps<{
  open: boolean
  mode: 'create' | 'edit'
  kind: 'profile' | 'project'
  initialName?: string
  initialContent?: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  save: [name: string, content: string]
}>()

const name = ref('')
const content = ref('')

watch(() => props.open, (open) => {
  if (open) {
    name.value = props.initialName ?? ''
    content.value = props.initialContent ?? ''
  }
})

function save() {
  if (!name.value.trim()) {
    return
  }
  emit('save', name.value.trim(), content.value)
  emit('update:open', false)
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="sm:max-w-md border-primary/20 bg-card">
      <DialogHeader>
        <DialogTitle class="text-primary font-mono">
          // {{ mode === 'create' ? 'new' : 'edit' }}_{{ kind }}
        </DialogTitle>
        <DialogDescription class="text-muted-foreground">
          {{ mode === 'create' ? 'Create' : 'Edit' }} a {{ kind }} memory file.
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4 py-2">
        <div class="space-y-2">
          <Label for="mem-name">Name</Label>
          <Input
            id="mem-name"
            v-model="name"
            :readonly="mode === 'edit'"
            placeholder="e.g. default"
            :class="mode === 'edit' ? 'opacity-60' : ''"
          />
        </div>

        <div class="space-y-2">
          <Label for="mem-content">Content (markdown)</Label>
          <Textarea
            id="mem-content"
            v-model="content"
            :rows="12"
            placeholder="Write markdown content..."
            class="resize-none font-mono text-xs"
          />
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="emit('update:open', false)">cancel</Button>
        <Button @click="save" :disabled="!name.trim()">save</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
