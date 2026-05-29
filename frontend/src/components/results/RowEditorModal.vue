<script setup>
import { VueMonacoEditor } from '@guolao/vue-monaco-editor'
import { NAlert, NButton, NFlex, NForm, NFormItem, NInput, NInputNumber, NModal, NSwitch, NText } from 'naive-ui'
import { computed, nextTick, ref, watch } from 'vue'
import { buildEditorFieldsFromValue, coerceEditorValue } from '@/lib/rowEditor'

const props = defineProps({
  show: Boolean,
  operation: String,
  row: Object,
  fields: {
    type: Array,
    default: () => [],
  },
  focusField: {
    type: String,
    default: '',
  },
  filter: String,
  source: String,
})
const emit = defineEmits(['update:show', 'submit', 'cancel'])

const visible = computed({
  get: () => props.show,
  set: val => emit('update:show', val),
})

const fieldRefs = new Map()
const jsonEditorRefs = new Map()
const localValues = ref({})
const localFilter = ref(props.filter || '')
const localSource = ref(props.source || '')
const jsonDrafts = ref({})
const jsonErrors = ref({})

const editorFields = computed(() => {
  if (Array.isArray(props.fields) && props.fields.length > 0)
    return props.fields
  return buildEditorFieldsFromValue(props.row ? { ...props.row } : {})
})

const modalTitle = computed(() => props.operation === 'delete' ? 'Confirm Delete' : 'Edit Row')

const monacoOptions = {
  automaticLayout: true,
  minimap: { enabled: false },
  fontSize: 12,
  scrollBeyondLastLine: false,
  wordWrap: 'on',
  fixedOverflowWidgets: true,
  lineNumbers: 'off',
}

function setFieldRef(key, el) {
  if (el)
    fieldRefs.set(key, el)
  else
    fieldRefs.delete(key)
}

function setJsonEditorRef(key, payload) {
  if (payload)
    jsonEditorRefs.set(key, payload)
  else
    jsonEditorRefs.delete(key)
}

function jsonModelPath(key) {
  const source = encodeURIComponent(props.source || 'row-editor')
  const filter = encodeURIComponent(props.filter || 'no-filter')
  const field = encodeURIComponent(key)
  return `inmemory://querybox/${source}/${filter}/${field}.json`
}

function jsonEditorKey(key) {
  return `${jsonModelPath(key)}:${props.show ? 'open' : 'closed'}`
}

function toJsonText(value) {
  if (value === null || value === undefined)
    return ''
  return JSON.stringify(value, null, 2)
}

function initState() {
  localValues.value = props.row ? { ...props.row } : {}
  localFilter.value = props.filter || ''
  localSource.value = props.source || ''
  jsonDrafts.value = {}
  jsonErrors.value = {}

  editorFields.value.forEach((field) => {
    localValues.value[field.key] = coerceEditorValue(field, localValues.value[field.key])
    if (field.kind === 'json')
      jsonDrafts.value[field.key] = toJsonText(localValues.value[field.key])
  })
}

function focusConfiguredField() {
  if (!props.focusField)
    return
  const jsonEditor = jsonEditorRefs.get(props.focusField)?.editor
  if (jsonEditor?.focus) {
    jsonEditor.focus()
    return
  }
  const target = fieldRefs.get(props.focusField)
  if (target?.focus)
    target.focus()
}

function syncJsonEditorModel(key) {
  const payload = jsonEditorRefs.get(key)
  if (!payload)
    return
  const { editor, monaco } = payload
  const model = editor?.getModel?.()
  if (!model || !monaco?.editor)
    return
  monaco.editor.setModelLanguage(model, 'json')
  nextTick(() => editor.layout?.())
}

watch(() => props.show, async (val) => {
  if (val) {
    initState()
    await nextTick()
    editorFields.value
      .filter(field => field.kind === 'json')
      .forEach(field => syncJsonEditorModel(field.key))
    focusConfiguredField()
  }
})

function updateJsonDraft(key, nextValue) {
  jsonDrafts.value[key] = nextValue
  if (!nextValue.trim()) {
    localValues.value[key] = null
    jsonErrors.value[key] = ''
    return
  }
  try {
    localValues.value[key] = JSON.parse(nextValue)
    jsonErrors.value[key] = ''
  }
  catch (error) {
    jsonErrors.value[key] = error instanceof Error ? error.message : 'Invalid JSON'
  }
}

function beautifyJson(key) {
  const source = jsonDrafts.value[key]
  if (!source?.trim()) {
    jsonDrafts.value[key] = ''
    jsonErrors.value[key] = ''
    localValues.value[key] = null
    return
  }
  try {
    const parsed = JSON.parse(source)
    jsonDrafts.value[key] = JSON.stringify(parsed, null, 2)
    localValues.value[key] = parsed
    jsonErrors.value[key] = ''
  }
  catch (error) {
    jsonErrors.value[key] = error instanceof Error ? error.message : 'Invalid JSON'
  }
}

function normalizeSubmitValues() {
  const nextValues = { ...localValues.value }
  for (const field of editorFields.value) {
    if (field.kind !== 'json')
      continue
    const source = jsonDrafts.value[field.key] ?? ''
    if (!source.trim()) {
      nextValues[field.key] = null
      continue
    }
    try {
      const parsed = JSON.parse(source)
      nextValues[field.key] = field.serializeJsonAsString
        ? JSON.stringify(parsed, null, 2)
        : parsed
      jsonErrors.value[field.key] = ''
    }
    catch (error) {
      jsonErrors.value[field.key] = error instanceof Error ? error.message : 'Invalid JSON'
    }
  }
  return nextValues
}

function handleSubmit() {
  const nextValues = normalizeSubmitValues()
  const hasJsonError = Object.values(jsonErrors.value).some(Boolean)
  if (hasJsonError)
    return

  emit('submit', {
    operation: props.operation,
    source: localSource.value,
    values: nextValues,
    filter: localFilter.value,
  })
  emit('update:show', false)
}

function handleCancel() {
  emit('update:show', false)
  emit('cancel')
}

function getNumberStep(field) {
  return field.numericMode === 'integer' ? 1 : 0.1
}

function handleJsonMount(key, editor, monaco) {
  setJsonEditorRef(key, { editor, monaco })
  syncJsonEditorModel(key)
}
</script>

<template>
  <NModal v-model:show="visible" preset="card" :style="{ width: '760px', maxWidth: '96vw' }" :title="modalTitle" closable>
    <div v-if="props.operation === 'update'">
      <NForm>
        <NFormItem v-for="field in editorFields" :key="field.key" :label="field.label || field.key">
          <div class="flex w-full flex-col gap-2">
            <NText v-if="field.rawType" depth="3" class="text-xs">
              {{ field.rawType }}
            </NText>

            <NInputNumber
              v-if="field.kind === 'number'"
              :ref="el => setFieldRef(field.key, el)"
              v-model:value="localValues[field.key]"
              class="w-full"
              :step="getNumberStep(field)"
              :precision="field.numericMode === 'integer' ? 0 : undefined"
            />

            <NSwitch
              v-else-if="field.kind === 'boolean'"
              :ref="el => setFieldRef(field.key, el)"
              v-model:value="localValues[field.key]"
            />

            <template v-else-if="field.kind === 'json'">
              <div class="json-editor-shell">
                <VueMonacoEditor
                  :key="jsonEditorKey(field.key)"
                  :ref="el => setFieldRef(field.key, el)"
                  v-model:value="jsonDrafts[field.key]"
                  :path="jsonModelPath(field.key)"
                  language="json"
                  default-language="json"
                  theme="vs-light"
                  :options="monacoOptions"
                  width="100%"
                  height="240px"
                  class-name="json-editor"
                  @mount="(editor, monaco) => handleJsonMount(field.key, editor, monaco)"
                  @update:value="value => updateJsonDraft(field.key, value ?? '')"
                />
              </div>
              <NFlex justify="space-between" align="center">
                <NButton size="small" secondary @click="beautifyJson(field.key)">
                  Beautify JSON
                </NButton>
                <NText depth="3" class="text-xs">
                  JSON must be valid before save
                </NText>
              </NFlex>
              <NAlert v-if="jsonErrors[field.key]" type="error" :show-icon="false">
                {{ jsonErrors[field.key] }}
              </NAlert>
            </template>

            <NInput
              v-else-if="field.kind === 'textarea'"
              :ref="el => setFieldRef(field.key, el)"
              v-model:value="localValues[field.key]"
              type="textarea"
              :autosize="{ minRows: 3, maxRows: 8 }"
            />

            <NInput
              v-else
              :ref="el => setFieldRef(field.key, el)"
              v-model:value="localValues[field.key]"
            />
          </div>
        </NFormItem>
      </NForm>
    </div>
    <div v-else>
      <p>Are you sure you want to delete this row?</p>
    </div>
    <template #footer>
      <NFlex justify="space-between" align="center">
        <NButton class="w-28" quaternary @click="handleCancel">
          Cancel
        </NButton>
        <NButton class="w-28" type="primary" @click="handleSubmit">
          {{ props.operation === 'delete' ? 'Delete' : 'Save' }}
        </NButton>
      </NFlex>
    </template>
  </NModal>
</template>

<style scoped>
.json-editor-shell {
  min-height: 220px;
  border: 1px solid var(--n-border-color, #e5e7eb);
  border-radius: 6px;
  overflow: hidden;
}

:deep(.json-editor) {
  height: 240px;
}
</style>
