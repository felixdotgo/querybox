<script setup>
import { NButton, NIcon, NSpin, NTag } from 'naive-ui'
import { computed, onBeforeUnmount, onMounted, ref, toRef, watch } from 'vue'
import { GetCredential } from '@/bindings/github.com/felixdotgo/querybox/services/connectionservice'
import { ExecPlugin } from '@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager'
import { useResultSort } from '@/composables/useResultSort'
import { useRowEditorModal } from '@/composables/useRowEditorModal'
import { getDataTypeColor, Key, Pencil, Pin, Trash } from '@/lib/icons'
import RowEditorModal from './RowEditorModal.vue'

const props = defineProps({
  payload: { type: Object, required: true },
  schema: { type: Object, required: false },
  connection: { type: Object, required: false },
  capabilities: { type: Array, default: () => [] },
  query: { type: String, default: '' },
})

const emit = defineEmits(['mutated'])

// ─── sort ───────────────────────────────────────────────────────────────────

const sortDatabase = computed(() => {
  const name = props.schema?.name
  if (name && typeof name === 'string' && name.includes('.'))
    return name.split('.')[0]
  return null
})

const { sortStates, isSorting, sortedPayload, handleSorterChange, resetSort } = useResultSort({
  query: toRef(props, 'query'),
  connection: toRef(props, 'connection'),
  database: sortDatabase,
})

watch(() => props.payload, resetSort)

// ─── capabilities ────────────────────────────────────────────────────────────

const showActions = computed(() => props.capabilities.includes('mutate-row'))
const showEdit = computed(() => {
  if (!showActions.value) return false
  const hasSub = props.capabilities.includes('mutate-row::edit') || props.capabilities.includes('mutate-row::delete')
  return !hasSub || props.capabilities.includes('mutate-row::edit')
})
const showDelete = computed(() => {
  if (!showActions.value) return false
  const hasSub = props.capabilities.includes('mutate-row::edit') || props.capabilities.includes('mutate-row::delete')
  return !hasSub || props.capabilities.includes('mutate-row::delete')
})

// ─── column definitions ──────────────────────────────────────────────────────

const COL_MIN_WIDTH = 120
const COL_CHAR_WIDTH = 9
// Extra space in header cell: pin button (20) + sort indicator (14) + padding (16) + buffer (8)
const COL_HEADER_BASE_EXTRA = 58

const pinnedColumns = ref([])

function togglePin(key) {
  const idx = pinnedColumns.value.indexOf(key)
  pinnedColumns.value = idx !== -1
    ? pinnedColumns.value.filter(k => k !== key)
    : [...pinnedColumns.value, key]
}

const tableColumns = computed(() => {
  let cols = props.payload.columns || []
  if (!Array.isArray(cols)) cols = Array.from(cols)

  const built = cols.map((c, idx) => {
    const name = c.name || `col${idx}`
    let typeString = null
    let typeColor = null
    let isPK = false

    if (props.schema && Array.isArray(props.schema.columns)) {
      const meta = props.schema.columns.find(x => x.name === name)
      if (meta?.type) {
        typeString = meta.type.replace(/\(.*\)$/, '').trim()
        typeColor = getDataTypeColor(typeString)
      }
      if (meta?.primary_key) isPK = true
    }

    // Width must accommodate: name text + type badge + PK icon + sort indicator + pin + padding
    const typeExtra = typeString ? Math.max(50, typeString.length * 7 + 20) : 0
    const pkExtra = isPK ? 16 : 0
    const headerExtra = COL_HEADER_BASE_EXTRA + typeExtra + pkExtra
    const width = Math.max(COL_MIN_WIDTH, name.length * COL_CHAR_WIDTH + headerExtra)

    return {
      key: name,
      title: name,
      width,
      typeString,
      typeColor,
      isPK,
      isPinned: pinnedColumns.value.includes(name),
      sortOrder: sortStates.value.get(name) ?? false,
    }
  })

  const pinned = pinnedColumns.value.map(k => built.find(c => c.key === k)).filter(Boolean)
  const unpinned = built.filter(c => !pinnedColumns.value.includes(c.key))
  return [...pinned, ...unpinned]
})

const totalWidth = computed(() =>
  tableColumns.value.reduce((s, c) => s + c.width, 0),
)

// ─── row data ────────────────────────────────────────────────────────────────

const rowOverrides = ref(new Map())

const tableData = computed(() => {
  const source = sortedPayload.value || props.payload
  let cols = source.columns || []
  if (!Array.isArray(cols)) cols = Array.from(cols)
  let rows = source.rows || []
  if (!Array.isArray(rows)) rows = Array.from(rows)

  return rows.map((r, rowIdx) => {
    const obj = { key: rowIdx }
    let vals = []
    if (r) {
      if (Array.isArray(r.values)) vals = r.values
      else if (Array.isArray(r.Values)) vals = r.Values
      else if (typeof r.getValues === 'function') vals = r.getValues()
    }
    ;(vals || []).forEach((v, i) => {
      const colName = (cols[i]?.name) ? cols[i].name : `col${i}`
      obj[colName] = v
    })
    const overrides = rowOverrides.value.get(rowIdx)
    if (overrides) Object.assign(obj, overrides)
    return obj
  })
})

// ─── virtual scroll ──────────────────────────────────────────────────────────

const ROW_HEIGHT = 33
const BUFFER_ROWS = 5

const bodyRef = ref(null)
const scrollTop = ref(0)
const viewportHeight = ref(400)

const startIndex = computed(() =>
  Math.max(0, Math.floor(scrollTop.value / ROW_HEIGHT) - BUFFER_ROWS),
)
const endIndex = computed(() =>
  Math.min(tableData.value.length, Math.ceil((scrollTop.value + viewportHeight.value) / ROW_HEIGHT) + BUFFER_ROWS),
)
const renderedRows = computed(() => tableData.value.slice(startIndex.value, endIndex.value))
const totalHeight = computed(() => tableData.value.length * ROW_HEIGHT)
const offsetY = computed(() => startIndex.value * ROW_HEIGHT)
// Visual offset for the fixed Actions panel (not inside the scroll container)
const actionsOffsetY = computed(() => offsetY.value - scrollTop.value)

function onScroll() {
  if (bodyRef.value) scrollTop.value = bodyRef.value.scrollTop
}

let ro
onMounted(() => {
  ro = new ResizeObserver(([entry]) => {
    const h = Math.floor(entry.contentRect.height)
    if (h > 0) viewportHeight.value = h
  })
  if (bodyRef.value) ro.observe(bodyRef.value)
})
onBeforeUnmount(() => ro?.disconnect())

// ─── column sort ─────────────────────────────────────────────────────────────

function handleColumnSort(col) {
  const current = col.sortOrder
  const order = current === false ? 'ascend' : current === 'ascend' ? 'descend' : false
  handleSorterChange({ columnKey: col.key, order, sorter: () => 0 })
}

// ─── row mutations ────────────────────────────────────────────────────────────

function escapeSqlValue(val) {
  if (val === null || val === undefined) return 'NULL'
  return String(val).replace(/'/g, "''")
}

function defaultFilterFor(row) {
  const parts = []
  for (const key in row) {
    if (key !== 'key') {
      const v = row[key]
      parts.push(v === null || v === undefined ? `${key} IS NULL` : `${key} = '${escapeSqlValue(v)}'`)
    }
  }
  return parts.join(' AND ')
}

function pkFilterFor(row) {
  const schemaCols = Array.isArray(props.schema?.columns) ? props.schema.columns : []
  const pkNames = schemaCols.filter(c => c.primary_key).map(c => c.name)
  if (pkNames.length === 0) return defaultFilterFor(row)
  const parts = []
  for (const key in row) {
    if (pkNames.includes(key)) {
      const v = row[key]
      parts.push(v === null || v === undefined ? `${key} IS NULL` : `${key} = '${escapeSqlValue(v)}'`)
    }
  }
  return parts.join(' AND ')
}

function sourceFrom() {
  return props.schema?.name ?? ''
}

function namedRow(row) {
  const { key: _key, ...rest } = row
  return rest
}

const { showEditor, editorOperation, editorRow, editorFilter, editorSource, openEditor, closeEditor, performMutation } = useRowEditorModal()
const editorRowKey = ref(null)

function handleEdit(row) {
  const named = namedRow(row)
  editorRowKey.value = row.key
  openEditor('update', named, sourceFrom(), pkFilterFor(named))
}

function handleDelete(row) {
  const named = namedRow(row)
  editorRowKey.value = row.key
  openEditor('delete', named, sourceFrom(), pkFilterFor(named))
}

async function refreshRow(rowKey, source, filter) {
  if (!props.connection?.driver_type) { emit('mutated'); return }
  try {
    const connMap = {}
    const cred = await GetCredential(props.connection.id)
    if (cred) connMap.credential_blob = cred
    if (source?.includes('.')) connMap.database = source.split('.')[0]
    const res = await ExecPlugin(props.connection.driver_type, connMap, `SELECT * FROM ${source} WHERE ${filter} LIMIT 1`, {})
    let pl = res?.result?.Payload ?? {}
    if (pl.Sql) pl = pl.Sql
    const rows = Array.isArray(pl.Rows) ? pl.Rows : []
    if (rows.length === 0) { emit('mutated'); return }
    const freshVals = rows[0].Values ?? rows[0].values ?? []
    const schemaCols = Array.isArray(props.payload.columns) ? props.payload.columns : []
    const patch = {}
    freshVals.forEach((v, i) => { patch[schemaCols[i]?.name ?? `col${i}`] = v })
    rowOverrides.value = new Map(rowOverrides.value).set(rowKey, patch)
  }
  catch { emit('mutated') }
}

async function handleMutation(params) {
  const capturedRowKey = editorRowKey.value
  await performMutation(props.connection, params, ({ operation, source, filter } = {}) => {
    if (operation === 'delete') emit('mutated')
    else refreshRow(capturedRowKey, source, filter)
  })
}
</script>

<template>
  <div class="relative flex h-full w-full overflow-hidden">
    <!-- Sorting overlay -->
    <div
      v-if="isSorting"
      class="absolute inset-0 z-20 flex items-center justify-center gap-2 bg-white/70"
    >
      <NSpin :size="20" />
      <span class="text-sm text-gray-500">Sorting...</span>
    </div>

    <!-- Scrollable data area -->
    <div
      ref="bodyRef"
      class="min-h-0 flex-1 overflow-auto pb-10"
      @scroll.passive="onScroll"
    >
      <div :style="{ minWidth: `${totalWidth}px` }">
        <!-- Sticky header -->
        <div class="sticky top-0 z-10 flex border-b border-gray-200 bg-slate-50 text-xs font-semibold text-gray-600">
          <div
            v-for="col in tableColumns"
            :key="col.key"
            class="flex h-8 shrink-0 cursor-pointer select-none items-center gap-1 border-r border-gray-200 px-2 hover:bg-slate-100"
            :style="{ width: `${col.width}px` }"
            :title="col.title"
            @click="handleColumnSort(col)"
          >
            <NIcon v-if="col.isPK" :size="12" class="shrink-0 text-yellow-400">
              <Key />
            </NIcon>
            <span class="flex-1 whitespace-nowrap">{{ col.title }}</span>
            <NTag
              v-if="col.typeString"
              size="tiny"
              :color="col.typeColor"
              round
              type="info"
              class="shrink-0"
            >
              {{ col.typeString }}
            </NTag>
            <span v-if="col.sortOrder === 'ascend'" class="shrink-0 text-blue-500">↑</span>
            <span v-else-if="col.sortOrder === 'descend'" class="shrink-0 text-blue-500">↓</span>
            <button
              class="pin-btn inline-flex shrink-0 cursor-pointer items-center justify-center rounded border-0 bg-transparent p-0.5"
              :class="{ 'is-pinned': col.isPinned }"
              :title="col.isPinned ? 'Unpin column' : 'Pin column'"
              @click.stop="togglePin(col.key)"
            >
              <NIcon :size="14">
                <Pin />
              </NIcon>
            </button>
          </div>
        </div>

        <!-- Virtual scroll body -->
        <div :style="{ height: `${totalHeight}px`, position: 'relative' }">
          <div :style="{ transform: `translateY(${offsetY}px)` }">
            <div
              v-for="(row, i) in renderedRows"
              :key="row.key"
              class="flex border-b border-gray-100 hover:bg-blue-50/40"
              :class="(startIndex + i) % 2 === 1 ? 'bg-gray-50/60' : 'bg-white'"
              :style="{ height: `${ROW_HEIGHT}px` }"
            >
              <div
                v-for="col in tableColumns"
                :key="col.key"
                class="flex shrink-0 items-center overflow-hidden border-r border-gray-100 px-2 text-xs"
                :style="{ width: `${col.width}px` }"
                :title="String(row[col.key] ?? '')"
              >
                <span class="truncate">{{ row[col.key] ?? '' }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Fixed Actions panel (pinned to the right, outside scroll area) -->
    <div
      v-if="showActions"
      class="flex w-[100px] shrink-0 flex-col border-l border-gray-200"
    >
      <!-- Header — height must match data header h-8 (32px) -->
      <div class="flex h-8 shrink-0 items-center justify-center border-b border-gray-200 bg-slate-50 text-xs font-semibold text-gray-600">
        Actions
      </div>
      <!-- Body — clips overflow, rows positioned by visual offset (offsetY - scrollTop) -->
      <div class="relative flex-1 overflow-hidden">
        <div :style="{ transform: `translateY(${actionsOffsetY}px)` }">
          <div
            v-for="(row, i) in renderedRows"
            :key="row.key"
            class="flex items-center justify-center gap-1 border-b border-gray-100"
            :class="(startIndex + i) % 2 === 1 ? 'bg-gray-50/60' : 'bg-white'"
            :style="{ height: `${ROW_HEIGHT}px` }"
          >
            <NButton
              v-if="showEdit"
              tertiary
              size="small"
              title="Edit row"
              @click="handleEdit(row)"
            >
              <template #icon>
                <NIcon :size="14">
                  <Pencil />
                </NIcon>
              </template>
            </NButton>
            <NButton
              v-if="showDelete"
              tertiary
              size="small"
              title="Delete row"
              @click="handleDelete(row)"
            >
              <template #icon>
                <NIcon :size="14">
                  <Trash />
                </NIcon>
              </template>
            </NButton>
          </div>
        </div>
      </div>
    </div>

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
.pin-btn {
  color: var(--n-title-text-color, #888);
  transition: color 0.15s, background-color 0.15s;
}
.pin-btn:hover {
  background-color: var(--n-border-color, #e8e8e8);
}
.pin-btn.is-pinned {
  color: var(--n-loading-color, #18a058);
}
</style>
