<script setup>
import { NButton, NIcon } from 'naive-ui'
import { computed, ref } from 'vue'
import { Pencil, Trash } from '@/lib/icons'
import JsonNode from './JsonNode.vue'
import RowEditorModal from './RowEditorModal.vue'
import { useRowEditorModal } from '@/composables/useRowEditorModal'

const props = defineProps({
  // Already-unwrapped document payload: either
  //   { document: <any> }            (legacy/single-document)
  //   { documents: [<any>, ...] }    (current proto)
  payload: {
    type: Object,
    required: true,
  },
  connection: {
    type: Object,
    required: false,
  },
  capabilities: {
    type: Array,
    default: () => [],
  },
})

const emit = defineEmits(['mutated'])

// Capability-gated visibility — backward-compat: "mutate-row" alone shows both buttons.
const showActions = computed(() => props.capabilities.includes('mutate-row'))
const showEdit = computed(() => {
  if (!showActions.value) return false
  const hasSub = props.capabilities.includes('mutate-row::edit') || props.capabilities.includes('mutate-row::delete')
  return !hasSub || props.capabilities.includes('mutate-row::edit')
})
const showDelete = computed(() => {
  if (!showActions.value) return false
  const hasSub = props.capabilities.includes('mutate-row::edit') || props.capabilities.includes('mutate-row::delete')
  return !hasSub || props.capabilities.includes('mutate-row::delete')
})

const {
  showEditor, editorOperation, editorRow: editorDoc, editorFilter, editorSource,
  openEditor, closeEditor, performMutation,
} = useRowEditorModal()

async function handleMutation(params) {
  await performMutation(props.connection, params, () => emit('mutated'))
}

// Normalised list of document payloads — always an array
const docs = computed(() => {
  if (props.payload.documents !== undefined) {
    return Array.isArray(props.payload.documents)
      ? props.payload.documents
      : Array.from(props.payload.documents)
  }
  if (props.payload.document !== undefined) {
    return [props.payload.document]
  }
  return []
})
</script>

<template>
  <div class="h-full w-full overflow-auto p-2">
    <template v-if="docs.length">
      <div
        v-for="(doc, idx) in docs"
        :key="idx"
        class="doc-row"
      >
        <div v-if="showEdit || showDelete" class="flex justify-end gap-2 mb-1">
          <NButton v-if="showEdit" size="small" tertiary title="Edit document" @click.stop.prevent="openEditor('update', doc)">
            <template #icon>
              <NIcon :size="16">
                <Pencil />
              </NIcon>
            </template>
          </NButton>
          <NButton v-if="showDelete" size="small" tertiary title="Delete document" @click.stop.prevent="openEditor('delete', doc)">
            <template #icon>
              <NIcon :size="16">
                <Trash />
              </NIcon>
            </template>
          </NButton>
        </div>
        <JsonNode :node-key="null" :value="doc" :depth="0" />
      </div>
    </template>
    <div v-else class="text-center text-gray-500 py-6 text-sm">
      (no documents)
    </div>
    <RowEditorModal
      v-model:show="showEditor"
      :operation="editorOperation"
      :row="editorDoc"
      :filter="editorFilter"
      :source="editorSource"
      @submit="handleMutation"
      @cancel="closeEditor"
    />
  </div>
</template>

<style scoped>
.doc-row {
  border: 1px solid var(--n-border-color, #e5e7eb);
  border-radius: 5px;
  margin-bottom: 6px;
  padding: 6px 10px;
  background-color: var(--n-color, #fff);
}
</style>
