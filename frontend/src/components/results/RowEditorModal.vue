<script setup>
import { NButton, NFlex, NForm, NFormItem, NInput, NModal } from 'naive-ui'
import { computed, ref, watch } from 'vue'

const props = defineProps({
  show: Boolean,
  operation: String,
  row: Object,
  filter: String,
  source: String,
})
const emit = defineEmits(['update:show', 'submit', 'cancel'])

// Writable computed so NModal's built-in close button (closable) correctly
// propagates update:show back to the parent rather than attempting a direct
// prop mutation (which is a Vue 3 readonly-prop violation and would leave the
// parent's showEditor ref stuck at true, preventing subsequent openings).
const visible = computed({
  get: () => props.show,
  set: val => emit('update:show', val),
})

// local copies so editing doesn't mutate props directly
const localValues = ref({})
const localFilter = ref(props.filter || '')
const localSource = ref(props.source || '')

watch(() => props.show, (val) => {
  if (val) {
    localValues.value = props.row ? { ...props.row } : {}
    localFilter.value = props.filter || ''
    localSource.value = props.source || ''
  }
})

function handleSubmit() {
  emit('submit', {
    operation: props.operation,
    source: localSource.value,
    values: localValues.value,
    filter: localFilter.value,
  })
  emit('update:show', false)
}
function handleCancel() {
  emit('update:show', false)
  emit('cancel')
}
</script>

<template>
  <NModal v-model:show="visible" preset="card" :style="{ width: '560px', maxWidth: '92vw' }" :title="props.operation === 'delete' ? 'Confirm Delete' : 'Edit Row'" closable>
    <div v-if="props.operation === 'update'">
      <NForm>
        <NFormItem v-for="(v, k) in localValues" :key="k" :label="k">
          <NInput v-model:value="localValues[k]" />
        </NFormItem>
      </NForm>
    </div>
    <div v-else>
      <p>Are you sure you want to delete this row?</p>
    </div>
    <template #footer>
      <NFlex justify="space-between" align="center">
        <NButton class="w-28" quaternary @click="handleCancel">
          Cancel
        </NButton>
        <NButton class="w-28" type="primary" @click="handleSubmit">
          {{ props.operation === 'delete' ? 'Delete' : 'Save' }}
        </NButton>
      </NFlex>
    </template>
  </NModal>
</template>
