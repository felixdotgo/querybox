<template>
  <div class="w-full overflow-auto">
    <template v-if="payload?.columns">
      <n-data-table
        :columns="tableColumns"
        :data="tableData"
        :row-key="rowKeyFunction"
        size="small"
        bordered
        striped
        scrollable
        class="w-full"
      />
    </template>
    <template v-else-if="payload?.document">
      <pre class="whitespace-pre-wrap">{{ payload.document | json }}</pre>
    </template>
    <template v-else-if="payload?.data">
      <!-- KV payload is normalized to { data: {...} } -->
      <n-descriptions bordered column="1">
        <n-descriptions-item
          v-for="(v, k) in payload.data || {}"
          :key="k"
          :label="k"
        >
          {{ v }}
        </n-descriptions-item>
      </n-descriptions>
    </template>
    <template v-else>
      <div class="text-gray-500">No Results</div>
    </template>
  </div>
</template>

<script setup>
import { computed } from "vue"

const props = defineProps({
  result: {
    type: Object,
    required: true,
  },
})

const payload = computed(() => {
  // unwrap potential ExecResult envelope produced by core-service
  const result = props.result || {}
  console.debug("ResultViewer received result prop", result)
  // The result coming from the backend may be:
  //   { columns:…, rows:… }            -- already unwrapped
  //   { sql: {…} }                      -- lowercase wrapper
  //   { Sql: {…} }                      -- capitalised wrapper
  //   PluginV1_ExecResult instance       -- JS class with Payload field
  // Unwrap repeatedly until we have the raw payload object.
  let r = result

  // if it's a protobuf class with Payload property, unwrap it
  if (r && typeof r === "object" && "Payload" in r) {
    r = r.Payload
  }

  if (r.sql) r = r.sql
  else if (r.Sql) r = r.Sql
  else if (r.document) r = r.document
  else if (r.Document) r = r.Document
  else if (r.kv) r = r.kv
  else if (r.Kv) r = r.Kv

  // second pass in case unwrapping produced another wrapper
  if (r && typeof r === "object") {
    if (r.sql) r = r.sql
    else if (r.Sql) r = r.Sql
    else if (r.document) r = r.document
    else if (r.Document) r = r.Document
    else if (r.kv) r = r.kv
    else if (r.Kv) r = r.Kv
  }

  console.debug("ResultViewer payload computed", r)
  return r
})

const tableColumns = computed(() => {
  // payload.value.columns may be a Vue proxy with numeric properties rather
  // than a true Array.  Use Array.from to normalise it.
  let cols = payload.value.columns || []
  if (!Array.isArray(cols)) {
    cols = Array.from(cols)
  }

  return cols.map((c, idx) => ({
    title: c.name || `col${idx}`,
    key: `col${idx}`,
    align: "left",
    ellipsis: true,
  }))
})

const tableData = computed(() => {
  let rows = payload.value.rows || []
  if (!Array.isArray(rows)) {
    rows = Array.from(rows)
  }

  return rows.map((r, rowIdx) => {
    const obj = { key: rowIdx }
    // support various shapes: r.values, r.Values, r.getValues()
    let vals = []
    if (r) {
      if (Array.isArray(r.values)) vals = r.values
      else if (Array.isArray(r.Values)) vals = r.Values
      else if (typeof r.getValues === "function") vals = r.getValues()
    }
    (vals || []).forEach((v, i) => {
      obj[`col${i}`] = v
    })
    return obj
  })
})

// helper for row-key prop
const rowKeyFunction = (row) => row && row.key
</script>
