<script setup>
import { computed } from 'vue'
import ResultViewerDocument from './ResultViewerDocument.vue'
import ResultViewerKeyValue from './ResultViewerKeyValue.vue'
import ResultViewerRdbms from './ResultViewerRdbms.vue'

defineEmits(['mutated'])

const props = defineProps({
  result: {
    type: Object,
    required: true,
  },
  schema: {
    type: Object,
    required: false,
  },
  // the connection object associated with this result tab; passed
  // through to viewers so they can perform mutations.
  connection: {
    type: Object,
    required: false,
  },
})

const payload = computed(() => {
  // Unwrap the ExecResult envelope produced by core-service.
  // The result coming from the backend may be:
  //   { columns:…, rows:… }            -- already unwrapped RDBMS
  //   { sql: {…} }                      -- lowercase sql wrapper
  //   { Sql: {…} }                      -- capitalised sql wrapper
  //   { document: {…} }                 -- document wrapper
  //   { kv: {…} }                       -- kv wrapper
  //   PluginV1_ExecResult instance       -- JS class with Payload field
  const result = props.result || {}
  console.debug('ResultViewer received result prop', result)

  let r = result

  // unwrap protobuf class envelope
  if (r && typeof r === 'object' && 'Payload' in r) {
    r = r.Payload
  }

  // first unwrap pass
  if (r.sql)
    r = r.sql
  else if (r.Sql)
    r = r.Sql
  else if (r.document)
    r = r.document
  else if (r.Document)
    r = r.Document
  else if (r.kv)
    r = r.kv
  else if (r.Kv)
    r = r.Kv

  // second pass in case unwrapping produced another wrapper
  if (r && typeof r === 'object') {
    if (r.sql)
      r = r.sql
    else if (r.Sql)
      r = r.Sql
    else if (r.document)
      r = r.document
    else if (r.Document)
      r = r.Document
    else if (r.kv)
      r = r.kv
    else if (r.Kv)
      r = r.Kv
  }

  console.debug('ResultViewer payload computed', r)
  return r
})

// Determine which sub-viewer to render based on the payload shape.
const viewType = computed(() => {
  const p = payload.value
  if (!p)
    return null
  if (p.columns)
    return 'rdbms'
  // proto defines DocumentResult as repeated Struct documents, not a
  // single "document" field.  previous code wrongly checked p.document and
  // therefore never activated when plugins returned multiple rows.
  if (p.documents !== undefined)
    return 'document'
  if (p.data !== undefined)
    return 'kv'
  return null
})
</script>

<template>
  <div class="h-full w-full overflow-hidden">
    <ResultViewerRdbms
      v-if="viewType === 'rdbms'"
      :payload="payload"
      :schema="props.schema"
      :connection="props.connection"
      @mutated="$emit('mutated')"
    />
    <ResultViewerDocument v-else-if="viewType === 'document'" :payload="payload" @mutated="$emit('mutated')" />
    <ResultViewerKeyValue v-else-if="viewType === 'kv'" :payload="payload" @mutated="$emit('mutated')" />
    <div v-else class="text-gray-500">
      No Results
    </div>
  </div>
</template>
