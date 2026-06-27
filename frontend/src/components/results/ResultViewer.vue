<script setup>
import { computed } from 'vue'
import { isEmptyExecPayload, resultViewType, unwrapExecPayload } from '@/lib/resultPayload'
import ResultViewerDocument from './ResultViewerDocument.vue'
import ResultViewerKeyValue from './ResultViewerKeyValue.vue'
import ResultViewerRdbms from './ResultViewerRdbms.vue'

const props = defineProps({
  result: {
    type: Object,
    required: true,
  },
  schema: {
    type: Object,
    required: false,
  },
  database: {
    type: String,
    required: false,
    default: null,
  },
  // the connection object associated with this result tab; passed
  // through to viewers so they can perform mutations.
  connection: {
    type: Object,
    required: false,
  },
  // plugin capabilities array forwarded from the tab context.
  capabilities: {
    type: Array,
    default: () => [],
  },
  query: {
    type: String,
    default: '',
  },
})

defineEmits(['mutated'])

const payload = computed(() => {
  return unwrapExecPayload(props.result)
})

// Determine which sub-viewer to render based on the payload shape.
const viewType = computed(() => {
  return resultViewType(payload.value)
})

const isEmptyPayload = computed(() => isEmptyExecPayload(payload.value))

const prettyPayload = computed(() => {
  const value = payload.value
  if (typeof value === 'string')
    return value
  try {
    return JSON.stringify(value, null, 2)
  }
  catch {
    return String(value)
  }
})
</script>

<template>
  <div class="h-full w-full overflow-hidden">
    <ResultViewerRdbms
      v-if="viewType === 'rdbms'"
      :payload="payload"
      :schema="props.schema"
      :database="props.database"
      :connection="props.connection"
      :capabilities="props.capabilities"
      :query="props.query"
      @mutated="$emit('mutated')"
    />
    <ResultViewerDocument
      v-else-if="viewType === 'document'"
      :payload="payload"
      :connection="props.connection"
      :capabilities="props.capabilities"
      :query="props.query"
      @mutated="$emit('mutated')"
    />
    <ResultViewerKeyValue
      v-else-if="viewType === 'kv'"
      :payload="payload"
      :connection="props.connection"
      :capabilities="props.capabilities"
      :query="props.query"
      @mutated="$emit('mutated')"
    />
    <div v-else-if="isEmptyPayload" class="flex h-full w-full items-center justify-center p-6 text-center text-sm text-gray-500">
      No data returned by this action
    </div>
    <div v-else class="h-full w-full overflow-auto p-4">
      <div class="mb-3 text-sm font-medium text-slate-700">
        Unsupported result payload
      </div>
      <pre class="overflow-auto rounded border border-slate-200 bg-slate-50 p-3 font-mono text-xs text-slate-700">{{ prettyPayload }}</pre>
    </div>
  </div>
</template>
