<script setup>
import { NButton, NIcon } from 'naive-ui'
import { computed, defineEmits, ref } from 'vue'
import { Pencil, Trash } from '@/lib/icons'
import RowEditorModal from './RowEditorModal.vue'
import { useRowEditorModal } from '@/composables/useRowEditorModal'

const props = defineProps({
  // Already-unwrapped KV payload: { data: { key: value, ... } }
  payload: {
    type: Object,
    required: true,
  },
  connection: {
    type: Object,
    required: false,
  },
})

const emit = defineEmits(['mutated'])

// Normalise: payload may be { data: {...} } or a flat object of k/v pairs.
const entries = computed(() => props.payload.data || props.payload || {})

const {
  showEditor, editorOperation, editorRow, editorFilter, editorSource,
  openEditor: _openEditor, closeEditor, performMutation,
} = useRowEditorModal()

function openEditor(op, row) {
  _openEditor(op, { ...row })
}

async function handleMutation(params) {
  await performMutation(props.connection, params, () => emit('mutated'))
}
</script>

<template>
  <n-descriptions bordered column="1">
    <n-descriptions-item
      v-for="(v, k) in entries"
      :key="k"
      :label="String(k)"
    >
      <div class="flex justify-end gap-2 mb-1">
        <NButton size="small" tertiary title="Edit entry" @click.stop.prevent="openEditor('update', { [k]: v })">
          <template #icon>
            <NIcon :size="16">
              <Pencil />
            </NIcon>
          </template>
        </NButton>
        <NButton size="small" tertiary title="Delete entry" @click.stop.prevent="openEditor('delete', { [k]: v })">
          <template #icon>
            <NIcon :size="16">
              <Trash />
            </NIcon>
          </template>
        </NButton>
      </div>
      {{ v }}
    </n-descriptions-item>
  </n-descriptions>
  <RowEditorModal
    v-model:show="showEditor"
    :operation="editorOperation"
    :row="editorRow"
    :filter="editorFilter"
    :source="editorSource"
    @submit="handleMutation"
    @cancel="closeEditor"
  />
</template>
