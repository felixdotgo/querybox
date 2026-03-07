<script setup>
import { VueMonacoEditor } from '@guolao/vue-monaco-editor'
import { onUnmounted, ref, watch } from 'vue'
import { useTabCompletion } from '@/composables/useTabCompletion'

const props = defineProps({
  modelValue: { type: String, default: '' },
  language: { type: String, default: 'sql' },
  theme: { type: String, default: 'vs-dark' },
  // full tab object — provides connection, node, capabilities context
  tab: { type: Object, default: null },
  // fallback connection when no tab is available
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
  quickSuggestions: true, // allow suggestions while typing
  suggestOnTriggerCharacters: true,
}

// ─── reactive tab ref for useTabCompletion ─────────────────────────────────
const tabRef = ref(props.tab)
watch(() => props.tab, (t) => { tabRef.value = t })

// @ts-expect-error: composable may lack typings
const completion = useTabCompletion(tabRef)

// ─── keyword / function / type maps ────────────────────────────────────────

const SQL_KEYWORDS = [
  'SELECT',
  'FROM',
  'WHERE',
  'INSERT',
  'INTO',
  'UPDATE',
  'DELETE',
  'SET',
  'JOIN',
  'LEFT JOIN',
  'RIGHT JOIN',
  'INNER JOIN',
  'OUTER JOIN',
  'FULL JOIN',
  'CROSS JOIN',
  'ON',
  'AND',
  'OR',
  'NOT',
  'IN',
  'IS',
  'NULL',
  'IS NULL',
  'IS NOT NULL',
  'LIKE',
  'BETWEEN',
  'EXISTS',
  'AS',
  'DISTINCT',
  'ALL',
  'ANY',
  'SOME',
  'CREATE',
  'DROP',
  'ALTER',
  'TABLE',
  'INDEX',
  'VIEW',
  'DATABASE',
  'SCHEMA',
  'TRUNCATE',
  'EXPLAIN',
  'ANALYZE',
  'BEGIN',
  'COMMIT',
  'ROLLBACK',
  'TRANSACTION',
  'GRANT',
  'REVOKE',
  'UNION',
  'UNION ALL',
  'INTERSECT',
  'EXCEPT',
  'ORDER BY',
  'GROUP BY',
  'HAVING',
  'LIMIT',
  'OFFSET',
  'CASE',
  'WHEN',
  'THEN',
  'ELSE',
  'END',
  'PRIMARY KEY',
  'FOREIGN KEY',
  'REFERENCES',
  'UNIQUE',
  'NOT NULL',
  'DEFAULT',
  'CHECK',
]

const PGSQL_EXTRA = [
  'RETURNING',
  'ILIKE',
  'SIMILAR TO',
  'OVERLAPS',
  'OVER',
  'PARTITION BY',
  'ROWS BETWEEN',
  'RANGE BETWEEN',
  'PRECEDING',
  'FOLLOWING',
  'CURRENT ROW',
  'RANK',
  'DENSE_RANK',
  'ROW_NUMBER',
  'NTILE',
  'LAG',
  'LEAD',
  'FIRST_VALUE',
  'LAST_VALUE',
  'WITH',
  'RECURSIVE',
  'MATERIALIZED',
  'LATERAL',
  'JSONB',
  'JSON',
  'ARRAY',
  'HSTORE',
  '->',
  '->>',
  '#>',
  '#>>',
  '@>',
  '<@',
  '?',
  '?|',
  '?&',
  '||',
  'COPY',
  'VACUUM',
  'REINDEX',
  'CLUSTER',
  'NOTIFY',
  'LISTEN',
  'DO',
  'DECLARE',
  'RAISE',
  'EXCEPTION',
]

const MYSQL_EXTRA = [
  'AUTO_INCREMENT',
  'UNSIGNED',
  'ZEROFILL',
  'ENGINE',
  'CHARSET',
  'COLLATE',
  'ON DUPLICATE KEY UPDATE',
  'INSERT IGNORE',
  'REPLACE INTO',
  'GROUP_CONCAT',
  'CONCAT_WS',
  'IFNULL',
  'NULLIF',
  'LOAD DATA',
  'INTO OUTFILE',
  'SHOW TABLES',
  'SHOW DATABASES',
  'SHOW COLUMNS',
  'DESCRIBE',
  'USE',
]

const SQLITE_EXTRA = [
  'PRAGMA',
  'ATTACH DATABASE',
  'DETACH DATABASE',
  'VACUUM',
  'REINDEX',
  'WITHOUT ROWID',
  'STRICT',
]

const AQL_KEYWORDS = [
  'FOR',
  'IN',
  'RETURN',
  'FILTER',
  'LET',
  'SORT',
  'COLLECT',
  'LIMIT',
  'DISTINCT',
  'INSERT',
  'UPDATE',
  'REPLACE',
  'REMOVE',
  'UPSERT',
  'WITH',
  'GRAPH',
  'SHORTEST_PATH',
  'K_SHORTEST_PATHS',
  'PRUNE',
  'LIKE',
  'NOT LIKE',
  'NOT IN',
  'OUTBOUND',
  'INBOUND',
  'ANY',
  'ASC',
  'DESC',
  'INTO',
]

const REDIS_COMMANDS = [
  'GET',
  'SET',
  'DEL',
  'EXISTS',
  'EXPIRE',
  'TTL',
  'PERSIST',
  'PTTL',
  'KEYS',
  'SCAN',
  'RANDOMKEY',
  'RENAME',
  'TYPE',
  'OBJECT',
  'INCR',
  'INCRBY',
  'DECR',
  'DECRBY',
  'APPEND',
  'MGET',
  'MSET',
  'MSETNX',
  'GETSET',
  'SETNX',
  'SETEX',
  'PSETEX',
  'HGET',
  'HSET',
  'HMGET',
  'HMSET',
  'HGETALL',
  'HDEL',
  'HEXISTS',
  'HKEYS',
  'HVALS',
  'HLEN',
  'LPUSH',
  'RPUSH',
  'LPOP',
  'RPOP',
  'LRANGE',
  'LLEN',
  'LINDEX',
  'LSET',
  'LINSERT',
  'SADD',
  'SREM',
  'SMEMBERS',
  'SCARD',
  'SISMEMBER',
  'SINTERSTORE',
  'SUNION',
  'ZADD',
  'ZREM',
  'ZRANGE',
  'ZRANGEBYSCORE',
  'ZRANK',
  'ZSCORE',
  'ZCARD',
  'ZINCRBY',
  'PUBLISH',
  'SUBSCRIBE',
  'PSUBSCRIBE',
  'MULTI',
  'EXEC',
  'DISCARD',
  'WATCH',
  'UNWATCH',
  'AUTH',
  'SELECT',
  'PING',
  'INFO',
  'CONFIG',
  'DEBUG',
  'FLUSHDB',
  'FLUSHALL',
]

const SQL_FUNCTIONS = [
  'COUNT',
  'SUM',
  'AVG',
  'MIN',
  'MAX',
  'COALESCE',
  'NULLIF',
  'GREATEST',
  'LEAST',
  'UPPER',
  'LOWER',
  'TRIM',
  'LTRIM',
  'RTRIM',
  'LENGTH',
  'SUBSTR',
  'SUBSTRING',
  'CONCAT',
  'REPLACE',
  'REGEXP_REPLACE',
  'SPLIT_PART',
  'POSITION',
  'TO_DATE',
  'TO_TIMESTAMP',
  'NOW',
  'CURRENT_DATE',
  'CURRENT_TIMESTAMP',
  'DATE_TRUNC',
  'DATE_PART',
  'EXTRACT',
  'AGE',
  'ROUND',
  'CEIL',
  'FLOOR',
  'ABS',
  'MOD',
  'POWER',
  'SQRT',
  'CAST',
  'CONVERT',
  'TRY_CAST',
  'ROW_NUMBER',
  'RANK',
  'DENSE_RANK',
  'NTILE',
  'LAG',
  'LEAD',
  'JSON_AGG',
  'JSONB_AGG',
  'JSON_BUILD_OBJECT',
  'JSON_EXTRACT',
  'JSON_VALUE',
  'ARRAY_AGG',
  'STRING_AGG',
  'GROUP_CONCAT',
  'GENERATE_SERIES',
  'UNNEST',
]

const SQL_TYPES = [
  'INTEGER',
  'INT',
  'BIGINT',
  'SMALLINT',
  'TINYINT',
  'NUMERIC',
  'DECIMAL',
  'FLOAT',
  'DOUBLE PRECISION',
  'REAL',
  'BOOLEAN',
  'BOOL',
  'VARCHAR',
  'TEXT',
  'CHAR',
  'NVARCHAR',
  'DATE',
  'TIME',
  'TIMESTAMP',
  'TIMESTAMPTZ',
  'INTERVAL',
  'BYTEA',
  'BLOB',
  'BINARY',
  'JSON',
  'JSONB',
  'UUID',
  'SERIAL',
  'BIGSERIAL',
  'SMALLSERIAL',
]

function getKeywordsForTab() {
  const lang = props.language
  const driver = tabRef.value?.context?.conn?.driver_type || ''
  if (driver === 'arangodb')
    return AQL_KEYWORDS
  if (driver === 'redis')
    return REDIS_COMMANDS
  const extras = { pgsql: PGSQL_EXTRA, mysql: MYSQL_EXTRA, sqlite: SQLITE_EXTRA }
  return [...SQL_KEYWORDS, ...(extras[lang] || [])]
}

// ─── context-aware position analysis ───────────────────────────────────────

function analyzeContext(model, position) {
  const driver = tabRef.value?.context?.conn?.driver_type || ''
  const lineText = model.getValueInRange({
    startLineNumber: position.lineNumber,
    startColumn: 1,
    endLineNumber: position.lineNumber,
    endColumn: position.column,
  })

  if (driver === 'redis' && /^\s*\w*$/.test(lineText))
    return { type: 'redis_command' }

  const qualifiedMatch = lineText.match(/(\w+)\.\s*$/)
  if (qualifiedMatch)
    return { type: 'qualified_column', qualifier: qualifiedMatch[1] }

  if (/\b(?:FROM|JOIN|INTO|UPDATE)\s+\w*$/i.test(lineText))
    return { type: 'table' }

  if (driver === 'arangodb' && /\bIN\s+\w*$/i.test(lineText))
    return { type: 'table' }

  if (/\b(?:SELECT|WHERE|SET|ON|BY|HAVING|AND|OR|NOT)\s+\w*$/i.test(lineText))
    return { type: 'column' }

  if (driver === 'arangodb' && /\b(?:FILTER|RETURN|SORT|LET)\s+\w*$/i.test(lineText))
    return { type: 'column' }

  return { type: 'generic' }
}

// ─── load schema on connection change ──────────────────────────────────────

watch(
  () => tabRef.value?.context?.conn || props.connection,
  async (conn) => {
    if (conn) {
      await completion.load(conn)
      completion.prefetchForPrimaryTable()
    }
  },
  { immediate: true },
)

function handleMount(editor, monaco) {
  editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, () => {
    emit('execute')
  })

  // register completion provider; will be recreated when language changes
  let provider = null
  const registerProvider = (language) => {
    if (provider) {
      provider.dispose()
      provider = null
    }

    provider = monaco.languages.registerCompletionItemProvider(language, {
      triggerCharacters: [' ', '.', '\n'],
      provideCompletionItems: async (model, position) => {
        // Guard: only respond for THIS editor's model
        if (model.uri.toString() !== editor.getModel()?.uri.toString())
          return { suggestions: [] }

        const suggestions = []
        const wordInfo = model.getWordUntilPosition(position)
        const range = new monaco.Range(
          position.lineNumber,
          wordInfo.startColumn,
          position.lineNumber,
          wordInfo.endColumn,
        )
        const K = monaco.languages.CompletionItemKind
        const ctx = analyzeContext(model, position)
        const primaryTbl = completion.primaryTable.value
        // ── qualified column: identifier.<cursor> ──────────────────────
        if (ctx.type === 'qualified_column') {
          const details = completion.getColumnDetails(ctx.qualifier)
          if (details.length > 0) {
            details.forEach(col => suggestions.push(buildColumnSuggestion(col, range, K, '00')))
          }
          else {
            const fields = await completion.getCompletionFields(ctx.qualifier)
            fields.forEach(f => suggestions.push({
              label: f.name,
              kind: K.Field,
              insertText: f.name,
              detail: f.type ? `field · ${f.type}` : 'field (sampled)',
              sortText: `00${f.name}`,
              range,
            }))
          }
          return { suggestions }
        }

        // ── table / collection names only ─────────────────────────────────
        if (ctx.type === 'table') {
          completion.getTableNames().forEach(name => suggestions.push({
            label: name,
            kind: K.Struct,
            insertText: name,
            detail: 'table / collection',
            sortText: `00${name}`,
            range,
          }))
          return { suggestions }
        }

        // ── Redis command only ────────────────────────────────────────────
        if (ctx.type === 'redis_command') {
          REDIS_COMMANDS.forEach(cmd => suggestions.push({
            label: cmd,
            kind: K.Keyword,
            insertText: cmd,
            detail: 'command',
            sortText: `00${cmd}`,
            range,
          }))
          return { suggestions }
        }
        // ── column context: primary table first, then others ─────────────
        if (ctx.type === 'column') {
          if (primaryTbl) {
            const details = completion.getColumnDetails(primaryTbl)
            if (details.length > 0) {
              details.forEach(col => suggestions.push(buildColumnSuggestion(col, range, K, '00')))
            }
            else {
              const fields = await completion.getCompletionFields(primaryTbl)
              fields.forEach(f => suggestions.push({
                label: f.name,
                kind: K.Field,
                insertText: f.name,
                detail: f.type ? `field · ${f.type}` : 'field (sampled)',
                sortText: `00${f.name}`,
                range,
              }))
            }
          }
          const allSchemas = completion.getAllSchemas()
          Object.entries(allSchemas).forEach(([tableName, schema]) => {
            if (tableName === primaryTbl)
              return
            if (schema && Array.isArray(schema.columns)) {
              schema.columns.forEach(col => suggestions.push(buildColumnSuggestion(col, range, K, '01')))
            }
          })
          completion.getTableNames().forEach(name => suggestions.push({
            label: name,
            kind: K.Struct,
            insertText: name,
            detail: 'table / collection',
            sortText: `02${name}`,
            range,
          }))
          return { suggestions }
        }

        // ── generic: keywords + functions + types + tables + primary cols ──
        const driver = tabRef.value?.context?.conn?.driver_type || ''

        if (driver === 'arangodb') {
          AQL_KEYWORDS.forEach(kw => suggestions.push({ label: kw, kind: K.Keyword, insertText: kw, detail: 'keyword', sortText: `99${kw}`, range }))
        }
        else if (driver === 'redis') {
          REDIS_COMMANDS.forEach(cmd => suggestions.push({ label: cmd, kind: K.Keyword, insertText: cmd, detail: 'command', sortText: `99${cmd}`, range }))
        }
        else {
          getKeywordsForTab().forEach(kw => suggestions.push({ label: kw, kind: K.Keyword, insertText: kw, detail: 'keyword', sortText: `99${kw}`, range }))
          SQL_FUNCTIONS.forEach(fn => suggestions.push({
            label: fn,
            kind: K.Function,
            insertText: `${fn}($1)`,
            insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
            detail: 'function',
            sortText: `98${fn}`,
            range,
          }))
          SQL_TYPES.forEach(tp => suggestions.push({ label: tp, kind: K.TypeParameter, insertText: tp, detail: 'type', sortText: `97${tp}`, range }))
        }

        if (primaryTbl) {
          const details = completion.getColumnDetails(primaryTbl)
          if (details.length > 0) {
            details.forEach(col => suggestions.push(buildColumnSuggestion(col, range, K, '00')))
          }
          else {
            const fields = await completion.getCompletionFields(primaryTbl)
            fields.forEach(f => suggestions.push({
              label: f.name,
              kind: K.Field,
              insertText: f.name,
              detail: f.type ? `field · ${f.type}` : 'field (sampled)',
              sortText: `00${f.name}`,
              range,
            }))
          }
        }

        completion.getTableNames().forEach((name) => {
          const schema = completion.getSchema(name)
          const colCount = schema?.columns?.length || 0
          const colPreview = schema?.columns?.slice(0, 10).map(c => c.name).join(', ') || ''
          suggestions.push({
            label: name,
            kind: K.Struct,
            insertText: name,
            detail: `table · ${colCount} col${colCount !== 1 ? 's' : ''}`,
            documentation: colPreview ? { value: `**Columns:** ${colPreview}${colCount > 10 ? ', …' : ''}` } : undefined,
            sortText: `02${name}`,
            range,
          })
        })
        return { suggestions }
      },
    })
  }

  let currentLanguage = props.language
  registerProvider(currentLanguage)
  watch(
    () => props.language,
    (lang) => {
      if (lang !== currentLanguage) {
        currentLanguage = lang
        registerProvider(lang)
      }
    },
  )

  watch(
    () => completion.nodes.value,
    (nodes) => {
      if (nodes && nodes.length)
        editor.trigger('completion', 'editor.action.triggerSuggest', {})
    },
  )

  onUnmounted(() => {
    if (provider) {
      provider.dispose()
      provider = null
    }
  })
}

// ─── helpers ───────────────────────────────────────────────────────────────

function buildColumnSuggestion(col, range, K, sortPrefix) {
  const parts = []
  if (col.type)
    parts.push(col.type)
  if (col.primary_key)
    parts.push('PK')
  else if (col.nullable === false)
    parts.push('NOT NULL')
  const detail = parts.length ? `column · ${parts.join(' · ')}` : 'column'
  return {
    label: col.name,
    kind: K.Field,
    insertText: col.name,
    detail,
    sortText: sortPrefix + col.name,
    range,
  }
}
</script>

<template>
  <div class="h-full border border-gray-200">
    <VueMonacoEditor
      v-model:value="code"
      :language="language"
      :theme="theme"
      :options="editorOptions"
      @mount="handleMount"
    />
  </div>
</template>
