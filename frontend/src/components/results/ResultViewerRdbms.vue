<script setup>
import { NButton, NIcon, NTag } from 'naive-ui'
import { computed, defineEmits, h, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRowEditorModal } from '@/composables/useRowEditorModal'
import { getDataTypeColor, Key, Pencil, Pin, Trash } from '@/lib/icons'
import RowEditorModal from './RowEditorModal.vue'

const props = defineProps({
  // Already-unwrapped RDBMS payload: { columns: [...], rows: [...] }
  payload: {
    type: Object,
    required: true,
  },
  schema: {
    type: Object,
    required: false,
  },
  connection: {
    type: Object,
    required: false,
  },
})

const emit = defineEmits(['mutated'])

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

// compute a default filter string equality comparison from a row object
function defaultFilterFor(row) {
  if (!row)
    return ''
  const parts = []
  for (const key in row) {
    if (key === 'key')
      continue
    const val = row[key]
    // simple quoting; frontend-sanitization is responsibility of plugin/driver
    parts.push(`${key} = '${val}'`)
  }
  return parts.join(' AND ')
}

const {
  showEditor,
  editorOperation,
  editorRow,
  editorFilter,
  editorSource,
  openEditor,
  closeEditor,
  performMutation,
} = useRowEditorModal(defaultFilterFor)

const tableColumns = computed(() => {
  let cols = props.payload.columns || []
  if (!Array.isArray(cols)) {
    cols = Array.from(cols)
  }

  const colMap = new Map()
  cols.forEach((c, idx) => {
    const name = c.name || `col${idx}`
    // try to annotate with schema metadata if available
    let display = name
    let meta = null
    let typeString = null
    let typeColor = null
    let keyIcon = null
    if (props.schema && Array.isArray(props.schema.columns)) {
      meta = props.schema.columns.find(x => x.name === name)
      if (meta) {
        display = name
        if (meta.type) {
          // strip parenthesized length/precision, e.g. varchar(14) -> varchar
          const base = meta.type.replace(/\(.*\)$/, '').trim()
          typeString = base
          typeColor = getDataTypeColor(base)
        }
        if (meta.primary_key) {
          keyIcon = Key
        }
      }
    }
    const key = `col${idx}`
    const isPinned = pinnedColumns.value.includes(key)
    const width = Math.max(COL_MIN_WIDTH, display.length * COL_CHAR_WIDTH + 24)

    colMap.set(key, {
      title: () =>
        h('div', { class: 'flex items-center gap-1 w-full' }, [
          h(
            'span',
            { class: 'flex-1 truncate flex items-center gap-1 ml-auto' },
            [
              h('span', {}, display),
              typeString
                ? h(
                    NTag,
                    {
                      size: 'tiny',
                      color: typeColor,
                      round: true,
                      class: 'datatype-badge',
                      type: 'info',
                    },
                    { default: () => typeString },
                  )
                : null,
              keyIcon
                ? h(NIcon, { size: 16, class: 'primary-key-icon' }, { default: () => h(keyIcon) })
                : null,
            ],
          ),
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
            [h(NIcon, { size: 18 }, { default: () => h(isPinned ? Pin : Pin) })],
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
  // add actions column at end
  const actionsCol = {
    className: 'shadow-md bg-slate-50 w-[120px]',
    title: 'Actions',
    key: 'actions',
    align: 'center',
    width: 120, // fixed pixel width
    minWidth: 120, // prevent shrinking
    resizable: false, // user may not adjust size
    fixed: 'right',
    render: row => h('div', { class: 'flex gap-1 justify-center' }, [
      h(NButton, {
        class: 'cursor-pointer',
        title: 'Edit row',
        onClick: (e) => { e.stopPropagation(); openEditor('update', row) },
        tertiary: true,
        size: 'small',
      }, h(NIcon, { size: 16 }, { default: () => h(Pencil) })),
      h(NButton, {
        class: 'cursor-pointer',
        title: 'Delete row',
        onClick: (e) => { e.stopPropagation(); openEditor('delete', row) },
        tertiary: true,
        size: 'small',
      }, h(NIcon, { size: 16 }, { default: () => h(Trash) })),
    ]),
  }
  return [...pinned, ...unpinned, actionsCol]
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

async function handleMutation(params) {
  await performMutation(props.connection, params, () => emit('mutated'))
}
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
      :bordered="false"
      :single-line="false"
      striped
      scrollable
      resizable
      class="w-full"
    />
    <RowEditorModal
      v-model:show="showEditor"
      :operation="editorOperation"
      :row="editorRow"
      :filter="editorFilter"
      :source="editorSource"
      @submit="handleMutation"
      @cancel="closeEditor"
    />
  </div>
</template>

<style scoped>
:deep(.n-data-table-resize-button) {
  right: -4px !important;
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

:deep(.primary-key-icon) {
  color: var(--n-loading-color, #fffb1f); /* green for PK */
  opacity: 0.85;
}
</style>
