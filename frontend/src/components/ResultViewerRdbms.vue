<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

const props = defineProps({
  // Already-unwrapped RDBMS payload: { columns: [...], rows: [...] }
  payload: {
    type: Object,
    required: true,
  },
})

const COL_MIN_WIDTH = 120
const COL_CHAR_WIDTH = 9 // approximate px per character for column title

const tableColumns = computed(() => {
  let cols = props.payload.columns || []
  if (!Array.isArray(cols)) {
    cols = Array.from(cols)
  }

  return cols.map((c, idx) => {
    const name = c.name || `col${idx}`
    const minWidth = Math.max(COL_MIN_WIDTH, name.length * COL_CHAR_WIDTH + 24)
    return {
      title: name,
      key: `col${idx}`,
      align: 'left',
      minWidth,
      ellipsis: { tooltip: true },
    }
  })
})

const scrollX = computed(() => {
  let total = 0
  for (const col of tableColumns.value) {
    total += col.minWidth
  }
  return total
})

// --- fill-height via ResizeObserver ---
const wrapperRef = ref(null)
const tableHeight = ref(400)

let ro
onMounted(() => {
  ro = new ResizeObserver(([entry]) => {
    tableHeight.value = Math.floor(entry.contentRect.height)
  })
  if (wrapperRef.value)
    ro.observe(wrapperRef.value)
})
onBeforeUnmount(() => ro?.disconnect())

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
      if (Array.isArray(r.values))
        vals = r.values
      else if (Array.isArray(r.Values))
        vals = r.Values
      else if (typeof r.getValues === 'function')
        vals = r.getValues()
    }
    ;(vals || []).forEach((v, i) => {
      obj[`col${i}`] = v
    })
    return obj
  })
})

const rowKeyFunction = row => row && row.key
</script>

<template>
  <div ref="wrapperRef" class="h-full w-full">
    <n-data-table
      :columns="tableColumns"
      :data="tableData"
      :row-key="rowKeyFunction"
      :scroll-x="scrollX"
      :max-height="tableHeight"
      size="small"
      bordered
      striped
      scrollable
      class="w-full"
    />
  </div>
</template>
