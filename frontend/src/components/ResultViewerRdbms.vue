<template>
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

<script setup>
import { computed } from "vue"

const props = defineProps({
  // Already-unwrapped RDBMS payload: { columns: [...], rows: [...] }
  payload: {
    type: Object,
    required: true,
  },
})

const tableColumns = computed(() => {
  // payload.columns may be a Vue proxy with numeric properties rather
  // than a true Array. Use Array.from to normalise it.
  let cols = props.payload.columns || []
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
  let rows = props.payload.rows || []
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
    ;(vals || []).forEach((v, i) => {
      obj[`col${i}`] = v
    })
    return obj
  })
})

const rowKeyFunction = (row) => row && row.key
</script>
