<script setup>
import { NButton, NForm, NFormItem, NInput, NModal } from 'naive-ui'
import { computed, ref, watch } from 'vue'

const props = defineProps({
  show: Boolean,
  operation: String,
  row: Object,
  filter: String,
  source: String,
})
const emit = defineEmits(['update:show', 'submit', 'cancel'])

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
  <NModal v-model:show="props.show" :title="props.operation === 'delete' ? 'Confirm Delete' : 'Edit Row'" closable>
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
    <div class="mt-4">
      <NForm>
        <NFormItem label="Source">
          <NInput v-model:value="localSource" placeholder="(optional)" />
        </NFormItem>
        <NFormItem label="Filter">
          <NInput v-model:value="localFilter" />
        </NFormItem>
      </NForm>
    </div>
    <template #footer>
      <NButton @click="handleCancel">
        Cancel
      </NButton>
      <NButton type="primary" @click="handleSubmit">
        {{ props.operation === 'delete' ? 'Delete' : 'Save' }}
      </NButton>
    </template>
  </NModal>
</template>
