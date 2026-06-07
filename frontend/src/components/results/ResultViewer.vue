<script setup>
import { computed } from 'vue'
import { resultViewType, unwrapExecPayload } from '@/lib/resultPayload'
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
    <div v-else class="h-full w-full p-4 text-sm text-gray-500">
      No supported result payload
    </div>
  </div>
</template>
