<template>
  <div class="h-full border border-gray-200">
    <vue-monaco-editor
      v-model:value="code"
      :language="language"
      :theme="theme"
      :options="editorOptions"
      @mount="handleMount"
    />
  </div>
</template>

<script setup>
import { ref, watch, onUnmounted } from 'vue'
import { VueMonacoEditor } from '@guolao/vue-monaco-editor'
import { useConnectionTree } from '@/composables/useConnectionTree'

const props = defineProps({
  modelValue: { type: String, default: '' },
  language: { type: String, default: 'sql' },
  theme: { type: String, default: 'vs-dark' },
  // connection context forwarded from workspace; may be null when no tab selected
  context: { type: Object, default: null },
  // fallback connection, e.g. the currently selected connection in workspace
  connection: { type: Object, default: null },
})

const emit = defineEmits(['update:modelValue', 'execute'])

const code = ref(props.modelValue)

watch(() => props.modelValue, (newVal) => {
  if (newVal !== code.value) {
    code.value = newVal
  }
})

watch(code, (newVal) => {
  emit('update:modelValue', newVal)
})

const editorOptions = {
  automaticLayout: true,
  minimap: { enabled: false },
  fontSize: 12,
  scrollBeyondLastLine: false,
  wordWrap: 'on',
  fixedOverflowWidgets: true,
  quickSuggestions: true,          // allow suggestions while typing
  suggestOnTriggerCharacters: true,
}

// track schema for current connection
// @ts-ignore: composable may lack typings
const { nodes: schemaNodesRef, load: loadSchema, getTableNames, getColumns } = useConnectionTree()

let schemaNodes = []

// simple keyword lists per driver (could be extended later)
const keywordMap = {
  pgsql: ['select','from','where','insert','update','delete','join','on','create','drop','alter'],
  mysql: ['select','from','where','insert','update','delete','join','on','create','drop','alter'],
  sql: ['select','from','where','insert','update','delete','join','on','create','drop','alter'],
  sqlite: ['select','from','where','insert','update','delete','join','on','create','drop','alter'],
}

// watch the incoming connection from either context or explicit prop
watch(
  () => props.context?.conn || props.connection,
  async (conn) => {
    console.log('QueryEditor observed connection change', conn)
    if (conn) {
      await loadSchema(conn)
    }
    schemaNodes = schemaNodesRef.value || []
    console.log('schemaNodes after connection update', schemaNodes.length)
    if (schemaNodes.length && editorInstance) {
      // trigger suggestion as soon as schema becomes available
      editorInstance.trigger('auto', 'editor.action.triggerSuggest', {})
    }
  },
  { immediate: true }
)

let editorInstance = null

const handleMount = (editor, monaco) => {
  editorInstance = editor
  editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, () => {
    emit('execute')
  })

  // automatically show suggestions on keystroke, but only once schema has data
  editor.onDidChangeModelContent(() => {
    if (schemaNodes && schemaNodes.length) {
      editor.trigger('auto', 'editor.action.triggerSuggest', {})
    }
  })

  // register completion provider; we'll recreate when language or schema changes
  let provider = null
  const registerProvider = () => {
    if (provider) provider.dispose()
    console.log('registering provider for language', props.language)
    provider = monaco.languages.registerCompletionItemProvider(props.language, {
      // we'll manually trigger suggest on each keystroke instead of relying
      // on built-in triggerCharacters
      provideCompletionItems: (model, position) => {
        const suggestions = []
        // compute the current word/range so we can assign it to each item
        const wordInfo = model.getWordUntilPosition(position)
        const prefix = wordInfo.word || ''
        const range = new monaco.Range(
          position.lineNumber,
          wordInfo.startColumn,
          position.lineNumber,
          wordInfo.endColumn
        )
        // debug current schema
        if (schemaNodes && schemaNodes.length) {
          console.log('completion provider invoked, schemaNodes count', schemaNodes.length,
                        'first labels', schemaNodes.slice(0,5).map(n=>n.label),
                        'sample structure', JSON.stringify(schemaNodes.slice(0,3), null, 2))
        } else {
          console.log('completion provider invoked, schemaNodes empty')
        }
        console.log('completion prefix word:', prefix, 'range', range)
        // include keywords first
        const keywords = keywordMap[props.language] || []
        keywords.forEach((kw) => {
          suggestions.push({
            label: kw,
            kind: monaco.languages.CompletionItemKind.Keyword,
            insertText: kw,
            range,
          })
        })
        // add all table/collection/view names from cache directly
        const tableNames = getTableNames()
        if (tableNames.length) {
          console.log('adding table names for suggestions', tableNames.slice(0,20))
          tableNames.forEach((t) => {
            suggestions.push({
              label: t,
              kind: monaco.languages.CompletionItemKind.Struct,
              insertText: t,
              filterText: t,
              range,
            })
          })
        }
        // include every node label (data-bearing or container) as fallback
        const gatherNames = (nodes) => {
          nodes.forEach((n) => {
            if (n && n.label) {
              suggestions.push({
                label: n.label,
                kind: monaco.languages.CompletionItemKind.Struct,
                insertText: n.label,
                filterText: n.label,
                range,
              })
            }
            if (Array.isArray(n.children)) gatherNames(n.children)
          })
        }
        gatherNames(schemaNodes)
        // handle nested prefix completion (db., db.table., etc.)
        const line = model.getLineContent(position.lineNumber).substr(0, position.column - 1)
        const parts = line.split(/\s+/)
        const last = parts[parts.length - 1]
        if (last && last.endsWith('.')) {
          const prefix = last.slice(0, -1) // strip trailing dot
          const path = prefix.split('.')
          // simple case: single identifier, offer its columns via helper
          if (path.length === 1) {
            const cols = getColumns(path[0])
            console.log('columns for', path[0], cols)
            cols.forEach((col) => {
              suggestions.push({
                label: col,
                kind: monaco.languages.CompletionItemKind.Field,
                insertText: col,
                filterText: col,
                range,
              })
            })
          } else {
            // fallback to recursive traversal
            let current = schemaNodes
            for (const part of path) {
              const found = current.find((n) => n.label === part)
              if (!found) {
                current = []
                break
              }
              current = Array.isArray(found.children) ? found.children : []
            }
            current.forEach((n) => {
              suggestions.push({
                label: n.label,
                kind: monaco.languages.CompletionItemKind.Field,
                insertText: n.label,
                filterText: n.label,
                range,
              })
            })
          }
        }
        console.log('suggestions generated', suggestions.map(s=>s.label).slice(0,20))
        return { suggestions }
      }
    })
  }

  // watch relevant reactive values and re-register provider
  watch(
    () => [props.language, schemaNodesRef.value],
    ([_lang, nodes]) => {
      schemaNodes = nodes || []
      registerProvider()
      // if we just populated the schema, trigger a suggestion pass so the user
      // immediately sees table names even if they’ve already typed.
      if (schemaNodes.length) {
        editor.trigger('auto', 'editor.action.triggerSuggest', {})
      }
    },
    { immediate: true }
  )

  onUnmounted(() => {
    provider && provider.dispose()
  })
}
</script>
