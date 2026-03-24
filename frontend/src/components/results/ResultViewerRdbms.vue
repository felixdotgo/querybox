<script setup>
import { NButton, NIcon, NSpin, NTag } from 'naive-ui'
import { computed, defineEmits, h, onBeforeUnmount, onMounted, ref, toRef, watch } from 'vue'
import { GetCredential } from '@/bindings/github.com/felixdotgo/querybox/services/connectionservice'
import { ExecPlugin } from '@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager'
import { useResultSort } from '@/composables/useResultSort'
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
  capabilities: {
    type: Array,
    default: () => [],
  },
  query: {
    type: String,
    default: '',
  },
})

const emit = defineEmits(['mutated'])

// Derive database name from schema (e.g. "employees.users" → "employees").
// Passed to useResultSort so the sort re-execution targets the correct DB.
const sortDatabase = computed(() => {
  const name = props.schema?.name
  if (name && typeof name === 'string' && name.includes('.'))
    return name.split('.')[0]
  return null
})

const {
  sortStates,
  isSorting,
  sortedPayload,
  handleSorterChange,
  resetSort,
} = useResultSort({
  query: toRef(props, 'query'),
  connection: toRef(props, 'connection'),
  database: sortDatabase,
})

watch(() => props.payload, resetSort)

// Prevent column resize from spuriously triggering the sorter handler.
// When the user drags a resize handle, the mouseup can bubble to the column
// header and fire @update:sorter.  We track resize intent and suppress the
// sorter event while it is active.
const isResizing = ref(false)
let resizeMousedownListener = null

function onSorterChange(state) {
  if (isResizing.value) return
  handleSorterChange(state)
}

// Derive which action buttons are permitted by the plugin's declared capabilities.
// Backward-compat: a plugin that declares only "mutate-row" (no sub-capabilities)
// is treated as supporting both edit and delete.
const showActions = computed(() => props.capabilities.includes('mutate-row'))
const showEdit = computed(() => {
  if (!showActions.value)
    return false
  const hasSub = props.capabilities.includes('mutate-row::edit') || props.capabilities.includes('mutate-row::delete')
  return !hasSub || props.capabilities.includes('mutate-row::edit')
})
const showDelete = computed(() => {
  if (!showActions.value)
    return false
  const hasSub = props.capabilities.includes('mutate-row::edit') || props.capabilities.includes('mutate-row::delete')
  return !hasSub || props.capabilities.includes('mutate-row::delete')
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

// Build a WHERE filter using only primary-key columns when schema provides
// them; falls back to all columns (via defaultFilterFor) when no PKs are known.
function pkFilterFor(row) {
  const schemaCols = (props.schema && Array.isArray(props.schema.columns)) ? props.schema.columns : []
  const pkNames = schemaCols.filter(c => c.primary_key).map(c => c.name)
  if (pkNames.length === 0)
    return defaultFilterFor(row)
  const parts = []
  for (const key in row) {
    if (!pkNames.includes(key))
      continue
    parts.push(`${key} = '${row[key]}'`)
  }
  return parts.join(' AND ')
}

// Derive the source table name from the schema prop (e.g. "employees.users").
function sourceFrom() {
  return (props.schema && props.schema.name) ? props.schema.name : ''
}
function namedRow(row) {
  const { key: _key, ...rest } = row
  return rest
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
} = useRowEditorModal()

// rowKey of the row currently open in the editor (used for targeted refresh)
const editorRowKey = ref(null)

// Override map: rowKey (integer index) → { col0: val, col1: val, … }
// Populated after a successful UPDATE to patch the row in-place.
const rowOverrides = ref(new Map())

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
    const key = name
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
      sorter: () => 0,
      sortOrder: sortStates.value.get(key) ?? false,
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
  // add actions column at end only when the plugin supports mutate-row
  if (!showActions.value)
    return [...pinned, ...unpinned]

  const actionsCol = {
    className: 'shadow-md bg-slate-50 w-[120px]',
    title: 'Actions',
    key: 'actions',
    align: 'center',
    width: 120,
    minWidth: 120,
    resizable: false,
    fixed: 'right',
    render: (row) => {
      const btns = []
      if (showEdit.value) {
        btns.push(h(NButton, {
          class: 'cursor-pointer',
          title: 'Edit row',
          onClick: (e) => {
            e.stopPropagation()
            const named = namedRow(row)
            editorRowKey.value = row.key
            openEditor('update', named, sourceFrom(), pkFilterFor(named))
          },
          tertiary: true,
          size: 'small',
        }, h(NIcon, { size: 16 }, { default: () => h(Pencil) })))
      }
      if (showDelete.value) {
        btns.push(h(NButton, {
          class: 'cursor-pointer',
          title: 'Delete row',
          onClick: (e) => {
            e.stopPropagation()
            const named = namedRow(row)
            editorRowKey.value = row.key
            openEditor('delete', named, sourceFrom(), pkFilterFor(named))
          },
          tertiary: true,
          size: 'small',
        }, h(NIcon, { size: 16 }, { default: () => h(Trash) })))
      }
      return h('div', { class: 'flex gap-1 justify-center' }, btns)
    },
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
  if (wrapperRef.value) {
    ro.observe(wrapperRef.value)
    resizeMousedownListener = (e) => {
      if (e.target.closest('.n-data-table-resize-button')) {
        isResizing.value = true
        window.addEventListener('mouseup', () => {
          // Small delay so the click event that fires after mouseup is still suppressed
          setTimeout(() => { isResizing.value = false }, 50)
        }, { once: true })
      }
    }
    wrapperRef.value.addEventListener('mousedown', resizeMousedownListener)
  }
})
onBeforeUnmount(() => {
  ro?.disconnect()
  if (wrapperRef.value && resizeMousedownListener)
    wrapperRef.value.removeEventListener('mousedown', resizeMousedownListener)
})

const tableData = computed(() => {
  const source = sortedPayload.value || props.payload
  let cols = source.columns || []
  if (!Array.isArray(cols))
    cols = Array.from(cols)
  let rows = source.rows || []
  if (!Array.isArray(rows))
    rows = Array.from(rows)

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
      const colName = (cols[i] && cols[i].name) ? cols[i].name : `col${i}`
      obj[colName] = v
    })
    // apply any in-place overrides from a targeted row refresh after UPDATE
    const overrides = rowOverrides.value.get(rowIdx)
    if (overrides)
      Object.assign(obj, overrides)
    return obj
  })
})

const rowKeyFunction = row => row && row.key

// refreshRow fetches a single updated row and patches it in-place so the
// result viewer shows the new values without a full tab reload.
async function refreshRow(rowKey, source, filter) {
  if (!props.connection || !props.connection.driver_type) {
    emit('mutated')
    return
  }
  try {
    const connMap = {}
    const cred = await GetCredential(props.connection.id)
    if (cred)
      connMap.credential_blob = cred
    // forward database prefix derived from the qualified source name
    if (source && source.includes('.')) {
      const dbName = source.split('.')[0]
      if (dbName)
        connMap.database = dbName
    }
    const selectSQL = `SELECT * FROM ${source} WHERE ${filter} LIMIT 1`
    const res = await ExecPlugin(props.connection.driver_type, connMap, selectSQL, {})
    // Unwrap the ExecPlugin response: result.Payload.Sql (uppercase, protojson)
    let payload = res?.result?.Payload ?? {}
    if (payload.Sql)
      payload = payload.Sql
    const rows = Array.isArray(payload.Rows) ? payload.Rows : []
    if (rows.length === 0) {
      // row no longer exists or fetch failed — fall back to full refresh
      emit('mutated')
      return
    }
    // map returned column values back to named column keys
    const freshVals = rows[0].Values ?? rows[0].values ?? []
    const schemaCols = Array.isArray(props.payload.columns) ? props.payload.columns : []
    const patch = {}
    freshVals.forEach((v, i) => {
      const colName = (schemaCols[i] && schemaCols[i].name) ? schemaCols[i].name : `col${i}`
      patch[colName] = v
    })
    rowOverrides.value = new Map(rowOverrides.value).set(rowKey, patch)
  }
  catch {
    // on any error fall back to full refresh
    emit('mutated')
  }
}

async function handleMutation(params) {
  const capturedRowKey = editorRowKey.value
  await performMutation(props.connection, params, ({ operation, source, filter } = {}) => {
    if (operation === 'delete') {
      emit('mutated')
    }
    else {
      // UPDATE: refresh only the affected row in-place
      refreshRow(capturedRowKey, source, filter)
    }
  })
}
</script>

<template>
  <div ref="wrapperRef" class="h-full w-full pb-10">
    <n-spin :show="isSorting" description="Sorting...">
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
        @update:sorter="onSorterChange"
      />
    </n-spin>
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
