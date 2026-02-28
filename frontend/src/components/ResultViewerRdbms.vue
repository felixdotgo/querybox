<script setup>
import { NIcon } from 'naive-ui'
import { computed, h, onBeforeUnmount, onMounted, ref } from 'vue'
import { Pin } from '@/lib/icons'

const props = defineProps({
  // Already-unwrapped RDBMS payload: { columns: [...], rows: [...] }
  payload: {
    type: Object,
    required: true,
  },
})

const COL_MIN_WIDTH = 120
const COL_CHAR_WIDTH = 9 // approximate px per character for column title

// Ordered array so pinned columns stay in pin-order on the left
const pinnedColumns = ref([])

function togglePin(key) {
  const idx = pinnedColumns.value.indexOf(key)
  if (idx !== -1) {
    pinnedColumns.value = pinnedColumns.value.filter(k => k !== key)
  }
  else {
    pinnedColumns.value = [...pinnedColumns.value, key]
  }
}

const tableColumns = computed(() => {
  let cols = props.payload.columns || []
  if (!Array.isArray(cols)) {
    cols = Array.from(cols)
  }

  const colMap = new Map()
  cols.forEach((c, idx) => {
    const name = c.name || `col${idx}`
    const key = `col${idx}`
    const isPinned = pinnedColumns.value.includes(key)
    const width = Math.max(COL_MIN_WIDTH, name.length * COL_CHAR_WIDTH + 24)

    colMap.set(key, {
      title: () =>
        h('div', { class: 'flex items-center gap-1 w-full' }, [
          h('span', { class: 'flex-1 truncate' }, name),
          h(
            'button',
            {
              class: [
                'inline-flex items-center justify-center p-0.5 border-0 bg-transparent cursor-pointer rounded shrink-0 transition-all duration-150 pin-btn',
                isPinned ? 'is-pinned' : '',
              ],
              title: isPinned ? 'Unpin column' : 'Pin column',
              onClick: (e) => {
                e.stopPropagation()
                togglePin(key)
              },
            },
            [h(NIcon, { size: 12 }, { default: () => h(isPinned ? Pin : Pin) })],
          ),
        ]),
      key,
      align: 'left',
      // fixed columns need explicit width, not just minWidth
      width,
      minWidth: width,
      resizable: true,
      ellipsis: { tooltip: true },
      fixed: isPinned ? 'left' : undefined,
    })
  })

  // Pinned columns first (in pin order), then the rest in original order
  const pinned = pinnedColumns.value.map(k => colMap.get(k)).filter(Boolean)
  const unpinned = [...colMap.values()].filter(c => !c.fixed)
  return [...pinned, ...unpinned]
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
  <div ref="wrapperRef" class="h-full w-full pb-10">
    <n-data-table
      :columns="tableColumns"
      :data="tableData"
      :row-key="rowKeyFunction"
      :scroll-x="scrollX"
      :max-height="tableHeight"
      :height-for-row="heightForRow"
      size="small"
      bordered
      striped
      scrollable
      resizable
      class="w-full"
    />
  </div>
</template>

<style scoped>
:deep(.n-data-table-td) {
  border-right: 1px solid var(--n-border-color);
}
:deep(.n-data-table-resize-button) {
  right: -2.5px !important;
  opacity: 1 !important;
}
:deep(.n-data-table-resize-button)::after {
  height: 100% !important;
  width: 1px !important;
}

/* CSS-variable colors and parent-hover trigger can't be expressed with Tailwind */
:deep(.pin-btn) {
  color: var(--n-title-text-color);
}
:deep(.pin-btn:hover) {
  background-color: var(--n-border-color);
  color: var(--n-title-text-color);
}
:deep(.pin-btn.is-pinned) {
  color: var(--n-loading-color, #18a058);
}
</style>
