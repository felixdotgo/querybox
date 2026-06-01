<script setup>
import { NButton, NIcon, NInput, NInputNumber, NSpin, NSwitch, NTag } from 'naive-ui'
import { computed, onBeforeUnmount, onMounted, ref, toRef, watch } from 'vue'
import { GetCredential } from '@/bindings/github.com/felixdotgo/querybox/services/connectionservice'
import { ExecPlugin } from '@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager'
import { useResultSort } from '@/composables/useResultSort'
import { useRowEditorModal } from '@/composables/useRowEditorModal'
import { getDataTypeColor, Key, Pencil, Pin, Trash } from '@/lib/icons'
import { buildEditorFieldsFromSchema, coerceEditorValue, getEditorField, isJsonEditorField } from '@/lib/rowEditor'
import RowEditorModal from './RowEditorModal.vue'

const props = defineProps({
  payload: { type: Object, required: true },
  schema: { type: Object, required: false },
  database: { type: String, required: false, default: null },
  connection: { type: Object, required: false },
  capabilities: { type: Array, default: () => [] },
  query: { type: String, default: '' },
})

const emit = defineEmits(['mutated'])

const { sortStates, isSorting, sortedPayload, handleSorterChange, resetSort } = useResultSort({
  query: toRef(props, 'query'),
  connection: toRef(props, 'connection'),
  database: toRef(props, 'database'),
})

watch(() => props.payload, resetSort)

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
const COL_CHAR_WIDTH = 9
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
  if (!Array.isArray(cols))
    cols = Array.from(cols)

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
      if (meta?.primary_key)
        isPK = true
    }

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

const gridTemplate = computed(() =>
  tableColumns.value.map(c => `${c.width}px`).join(' '),
)

const rowOverrides = ref(new Map())

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
      const colName = (cols[i]?.name) ? cols[i].name : `col${i}`
      obj[colName] = v
    })
    const overrides = rowOverrides.value.get(rowIdx)
    if (overrides)
      Object.assign(obj, overrides)
    return obj
  })
})

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
const actionsOffsetY = computed(() => offsetY.value - scrollTop.value)

let scrollRaf = 0
function onScroll() {
  if (scrollRaf)
    return
  scrollRaf = requestAnimationFrame(() => {
    scrollRaf = 0
    if (bodyRef.value)
      scrollTop.value = bodyRef.value.scrollTop
  })
}

let ro
onMounted(() => {
  ro = new ResizeObserver(([entry]) => {
    const h = Math.floor(entry.contentRect.height)
    if (h > 0)
      viewportHeight.value = h
  })
  if (bodyRef.value)
    ro.observe(bodyRef.value)
})
onBeforeUnmount(() => {
  ro?.disconnect()
  if (scrollRaf)
    cancelAnimationFrame(scrollRaf)
})

function handleColumnSort(col) {
  const current = col.sortOrder
  const order = current === false ? 'ascend' : current === 'ascend' ? 'descend' : false
  handleSorterChange({ columnKey: col.key, order, sorter: () => 0 })
}

function escapeSqlValue(val) {
  if (val === null || val === undefined)
    return 'NULL'
  return String(val).replace(/'/g, '\'\'')
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
  if (pkNames.length === 0)
    return defaultFilterFor(row)
  const parts = []
  for (const key in row) {
    if (pkNames.includes(key)) {
      const v = row[key]
      parts.push(v === null || v === undefined ? `${key} IS NULL` : `${key} = '${escapeSqlValue(v)}'`)
    }
  }
  return parts.join(' AND ')
}

function filterValuesFor(row) {
  const schemaCols = Array.isArray(props.schema?.columns) ? props.schema.columns : []
  const pkNames = schemaCols.filter(c => c.primary_key).map(c => c.name)
  const keys = pkNames.length > 0 ? pkNames : Object.keys(row).filter(key => key !== 'key')
  const filterValues = {}
  keys.forEach((key) => {
    if (Object.prototype.hasOwnProperty.call(row, key))
      filterValues[key] = row[key]
  })
  return filterValues
}

function sourceFrom() {
  return props.schema?.name ?? ''
}

function namedRow(row) {
  const { key: _key, ...rest } = row
  return rest
}

const schemaColumns = computed(() => Array.isArray(props.schema?.columns) ? props.schema.columns : [])

function buildRowFields(row) {
  return buildEditorFieldsFromSchema(row, schemaColumns.value)
}

function getFieldForCell(row, key) {
  return getEditorField(buildRowFields(namedRow(row)), key)
}

function normalizeDraft(rowValues) {
  const draft = { ...rowValues }
  buildRowFields(rowValues).forEach((field) => {
    draft[field.key] = coerceEditorValue(field, draft[field.key])
  })
  return draft
}

function formatCellValue(value) {
  if (value === null || value === undefined)
    return ''
  if (typeof value === 'object')
    return JSON.stringify(value)
  return String(value)
}

const { showEditor, editorOperation, editorRow, editorFilter, editorSource, editorFields, editorFocusField, openEditor, closeEditor, performMutation } = useRowEditorModal()
const editorRowKey = ref(null)
const editorFilterValues = ref({})
const editingRowKey = ref(null)
const inlineDraftValues = ref({})
const inlineOriginalFilter = ref('')
const inlineOriginalFilterValues = ref({})

function resetInlineEdit() {
  editingRowKey.value = null
  inlineDraftValues.value = {}
  inlineOriginalFilter.value = ''
  inlineOriginalFilterValues.value = {}
}

function beginInlineEdit(row) {
  const named = namedRow(row)
  editingRowKey.value = row.key
  inlineDraftValues.value = normalizeDraft(named)
  inlineOriginalFilter.value = pkFilterFor(named)
  inlineOriginalFilterValues.value = filterValuesFor(named)
}

function openTypedEditor(row, focusField = '') {
  const named = editingRowKey.value === row.key && Object.keys(inlineDraftValues.value).length > 0
    ? { ...inlineDraftValues.value }
    : normalizeDraft(namedRow(row))
  editorRowKey.value = row.key
  editorFilterValues.value = filterValuesFor(namedRow(row))
  openEditor('update', named, sourceFrom(), pkFilterFor(namedRow(row)), buildRowFields(named), focusField)
}

function handleEdit(row) {
  openTypedEditor(row)
}

function handleDelete(row) {
  const named = namedRow(row)
  editorRowKey.value = row.key
  editorFilterValues.value = filterValuesFor(named)
  openEditor('delete', named, sourceFrom(), pkFilterFor(named), buildRowFields(named))
}

function handleCellDoubleClick(row, columnKey) {
  if (!showEdit.value)
    return
  const field = getFieldForCell(row, columnKey)
  if (isJsonEditorField(field)) {
    openTypedEditor(row, columnKey)
    return
  }
  beginInlineEdit(row)
}

function saveInlineEdit() {
  if (editingRowKey.value === null)
    return
  const capturedRowKey = editingRowKey.value
  const params = {
    operation: 'update',
    source: sourceFrom(),
    values: { ...inlineDraftValues.value },
    filter: inlineOriginalFilter.value,
    filterValues: { ...inlineOriginalFilterValues.value },
  }
  handleMutation(params, capturedRowKey)
}

async function refreshRow(rowKey, source, filter) {
  if (!props.connection?.driver_type) {
    emit('mutated')
    return
  }
  try {
    const connMap = {}
    const cred = await GetCredential(props.connection.id)
    if (cred)
      connMap.credential_blob = cred
    if (props.database)
      connMap.database = props.database
    const res = await ExecPlugin(props.connection.driver_type, connMap, `SELECT * FROM ${source} WHERE ${filter} LIMIT 1`, {})
    let pl = res?.result?.Payload ?? {}
    if (pl.Sql)
      pl = pl.Sql
    const rows = Array.isArray(pl.Rows) ? pl.Rows : []
    if (rows.length === 0) {
      emit('mutated')
      return
    }
    const freshVals = rows[0].Values ?? rows[0].values ?? []
    const schemaCols = Array.isArray(props.payload.columns) ? props.payload.columns : []
    const patch = {}
    freshVals.forEach((v, i) => { patch[schemaCols[i]?.name ?? `col${i}`] = v })
    rowOverrides.value = new Map(rowOverrides.value).set(rowKey, patch)
  }
  catch {
    emit('mutated')
  }
}

async function handleMutation(params, forcedRowKey = null) {
  const capturedRowKey = forcedRowKey ?? editorRowKey.value
  const mutationParams = {
    ...params,
    filterValues: params.filterValues || { ...editorFilterValues.value },
  }
  await performMutation(props.connection, mutationParams, props.database, ({ operation, source, filter } = {}) => {
    resetInlineEdit()
    editorFilterValues.value = {}
    if (operation === 'delete')
      emit('mutated')
    else refreshRow(capturedRowKey, source, filter)
  })
}

function isEditingRow(row) {
  return editingRowKey.value === row.key
}
</script>

<template>
  <div class="relative flex h-full w-full overflow-hidden">
    <div
      v-if="isSorting"
      class="absolute inset-0 z-20 flex items-center justify-center gap-2 bg-white/70"
    >
      <NSpin :size="20" />
      <span class="text-sm text-gray-500">Sorting...</span>
    </div>

    <div
      ref="bodyRef"
      class="min-h-0 flex-1 overflow-auto pb-10"
      @scroll.passive="onScroll"
    >
      <div :style="{ 'minWidth': `${totalWidth}px`, '--grid-cols': gridTemplate, '--row-h': `${ROW_HEIGHT}px` }">
        <div class="header-row sticky top-0 z-10 border-b border-gray-200 bg-slate-50 text-xs font-semibold text-gray-600">
          <div
            v-for="col in tableColumns"
            :key="col.key"
            class="flex h-8 cursor-pointer select-none items-center gap-1 border-r border-gray-200 px-2 hover:bg-slate-100"
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

        <div :style="{ height: `${totalHeight}px`, position: 'relative' }">
          <div :style="{ transform: `translateY(${offsetY}px)` }">
            <div
              v-for="row in renderedRows"
              :key="row.key"
              class="data-row border-b border-gray-100 hover:bg-blue-50/40"
              :class="row.key % 2 === 1 ? 'bg-gray-50/60' : 'bg-white'"
            >
              <div
                v-for="col in tableColumns"
                :key="col.key"
                class="flex items-center overflow-hidden border-r border-gray-100 px-2 text-xs"
                :title="formatCellValue(row[col.key])"
                @dblclick.stop="handleCellDoubleClick(row, col.key)"
              >
                <NInputNumber
                  v-if="isEditingRow(row) && getFieldForCell(row, col.key)?.kind === 'number'"
                  v-model:value="inlineDraftValues[col.key]"
                  class="w-full"
                  size="small"
                  :step="getFieldForCell(row, col.key)?.numericMode === 'integer' ? 1 : 0.1"
                  :precision="getFieldForCell(row, col.key)?.numericMode === 'integer' ? 0 : undefined"
                />
                <NSwitch
                  v-else-if="isEditingRow(row) && getFieldForCell(row, col.key)?.kind === 'boolean'"
                  v-model:value="inlineDraftValues[col.key]"
                  size="small"
                />
                <div v-else-if="isEditingRow(row) && getFieldForCell(row, col.key)?.kind === 'json'" class="flex w-full items-center justify-between gap-2">
                  <span class="truncate text-gray-500">JSON value</span>
                  <NButton size="tiny" secondary @click.stop="openTypedEditor(row, col.key)">
                    Edit JSON
                  </NButton>
                </div>
                <NInput
                  v-else-if="isEditingRow(row)"
                  v-model:value="inlineDraftValues[col.key]"
                  size="small"
                />
                <span v-else class="truncate">{{ formatCellValue(row[col.key]) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div
      v-if="showActions"
      class="flex w-[132px] shrink-0 flex-col border-l border-gray-200"
    >
      <div class="flex h-8 shrink-0 items-center justify-center border-b border-gray-200 bg-slate-50 text-xs font-semibold text-gray-600">
        Actions
      </div>
      <div class="relative flex-1 overflow-hidden">
        <div :style="{ transform: `translateY(${actionsOffsetY}px)` }">
          <div
            v-for="row in renderedRows"
            :key="row.key"
            class="flex items-center justify-center gap-1 border-b border-gray-100"
            :class="row.key % 2 === 1 ? 'bg-gray-50/60' : 'bg-white'"
            :style="{ height: `${ROW_HEIGHT}px` }"
          >
            <template v-if="isEditingRow(row)">
              <NButton tertiary size="small" type="primary" @click="saveInlineEdit">
                Save
              </NButton>
              <NButton tertiary size="small" @click="resetInlineEdit">
                Cancel
              </NButton>
            </template>
            <template v-else>
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
            </template>
          </div>
        </div>
      </div>
    </div>

    <RowEditorModal
      v-model:show="showEditor"
      :operation="editorOperation"
      :row="editorRow"
      :fields="editorFields"
      :focus-field="editorFocusField"
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
.header-row,
.data-row {
  display: grid;
  grid-template-columns: var(--grid-cols);
}
.data-row {
  height: var(--row-h);
}
</style>
