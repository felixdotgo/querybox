<script setup>
import { NButton, NIcon, useNotification } from 'naive-ui'
import { computed, onMounted, ref, toRef, watch } from 'vue'
import { ResultViewer } from '@/components/results'
import { useConnectionTree } from '@/composables/useConnectionTree'
import { Analytics, Play } from '@/lib/icons'
import QueryEditor from './QueryEditor.vue'
import TableStructureViewer from './TableStructureViewer.vue'
import WelcomeTab from './WelcomeTab.vue'

const props = defineProps({
  selectedConnection: { type: Object, default: null },
})

const emit = defineEmits(['tab-closed', 'active-connection-changed', 'refresh-tab'])

// allow lookup of cached schemas; provide selectedConnection ref so
// schema-related helpers know which connection to query.
const { getSchema, fetchSchema } = useConnectionTree(toRef(props, 'selectedConnection'))
const notification = useNotification()

const currentSchema = computed(() => {
  // eslint-disable-next-line ts/no-use-before-define
  const tab = tabs.value.find(t => t.key === activeTabKey.value)
  if (!tab || !tab.context || !tab.context.node)
    return null

  // node.key may include conn prefix ("<conn.id>:"), strip if present
  let key = tab.context.node.key
  if (key && typeof key === 'string' && tab.context?.conn && key.startsWith(`${tab.context.conn.id}:`)) {
    key = key.slice((`${tab.context.conn.id}:`).length)
  }

  // Deep tree plugins (e.g. PostgreSQL: db→schema→"Tables"→table) build
  // hierarchical keys like "mydb:public:public.Tables:public.users".
  // Extract only the last colon-separated segment so getSchema receives
  // the actual node key (e.g. "public.users") regardless of depth.
  const lastColon = key ? key.lastIndexOf(':') : -1
  if (lastColon !== -1) {
    key = key.slice(lastColon + 1)
  }

  if (!key || typeof key !== 'string')
    return null
  const schema = getSchema(key, tab.context.conn)
  console.debug('currentSchema lookup', key, 'conn', tab.context?.conn?.id, schema)
  return schema
})
const tabs = ref([])
const activeTabKey = ref('')

onMounted(() => {
  tabs.value = [{ key: '__welcome__', title: 'Welcome', type: 'welcome' }]
  activeTabKey.value = '__welcome__'
})

function supportsExplain(tab) {
  return !!(tab && tab.context && Array.isArray(tab.context.capabilities) && tab.context.capabilities.includes('explain-query'))
}

function handleExplain(tab) {
  if (!tab.context)
    return
  tab.loading = true
  tab.innerTab = 'explain'
  const explainContext = {
    ...tab.context,
    action: {
      ...tab.context.action,
      query: tab.query,
    },
    explain: true,
  }
  emit('refresh-tab', explainContext)
}

function getMonacoLanguage(driver) {
  if (!driver)
    return 'sql'
  const d = driver.toLowerCase()
  if (d.includes('postgres') || d.includes('psql'))
    return 'pgsql'
  if (d.includes('mysql'))
    return 'mysql'
  if (d.includes('sqlite'))
    return 'sql'
  if (d.includes('redis'))
    return 'redis'
  if (d.includes('arangodb'))
    return 'javascript' // AQL is not supported, javascript is close enough or use sql
  return 'sql'
}

// helper to derive a table name from a tab object (mirrors
// currentSchema computation logic).  returns null if no valid table.
function extractTableName(tab) {
  if (!tab || !tab.context || !tab.context.node)
    return null
  let key = tab.context.node.key
  if (key && typeof key === 'string' && key.startsWith(`${tab.context.conn?.id}:`)) {
    key = key.slice((`${tab.context.conn.id}:`).length)
  }
  // For deep-hierarchy plugins (e.g. PostgreSQL) the remaining key still
  // contains parent-path segments; keep only the last colon-segment.
  const lastColon = key ? key.lastIndexOf(':') : -1
  if (lastColon !== -1) {
    key = key.slice(lastColon + 1)
  }
  if (!key || typeof key !== 'string')
    return null
  return key
}

watch(activeTabKey, (key) => {
  console.debug('activeTabKey changed to', key)
  // tabKey format: conn.id + ":" + node.key — extract the connection ID
  const connId = key ? key.split(':')[0] : null
  emit('active-connection-changed', connId || null)

  // if the new tab targets a table for which we don't yet have schema,
  // kick off a background fetch. the composable will merge results and the
  // `currentSchema` watcher will flip us to the Structure page when data
  // arrives.
  const tab = tabs.value.find(t => t.key === key)
  const tbl = extractTableName(tab)
  console.debug('activeTabKey watcher table', tbl, 'cached?', tbl ? !!getSchema(tbl, tab.context?.conn) : null)
  if (tbl) {
    if (!getSchema(tbl, tab.context?.conn)) {
      console.debug('activeTabKey watcher invoking fetchSchema for', tbl, 'conn', tab.context?.conn?.id)
      fetchSchema(tbl, tab.context?.conn).catch(err => console.error('fetchSchema failed', err))
    }
    else {
      console.debug('schema already cached for', tbl, 'conn', tab.context?.conn?.id)
    }
  }
})

function openTab(title, result, error, tabKey, version, context) {
  // sanitize human title just in case it still contains a prefix
  const sanitize = t => (t ? t.split(':').pop() : t)
  title = sanitize(title)

  // `tabKey` is a stable identifier used internally to avoid opening
  // duplicate tabs. the human‑readable title shown on the tab is always
  // supplied separately (usually the node.key such as "db.table").
  // only when `tabKey` is absent do we fall back to the title, and as a
  // last resort we generate a random id.
  let key
  if (tabKey) {
    key = tabKey
  }
  else if (title) {
    key = title
  }
  else {
    key = `${Date.now()}-${Math.random().toString(36).slice(2)}`
  }

  // migration support for older tabs that used the title as the key.
  let existing = tabs.value.find(t => t.key === key)
  if (!existing && tabKey && title) {
    const alt = tabs.value.find(t => t.key === title)
    if (alt) {
      // promote alt to the new key. the old tab may have been created when
      // we used the title as the key; if that title contained a connection
      // hash we no longer want to show it, so update the stored title too.
      alt.key = key
      alt.title = title
      existing = alt
    }
  }

  // ignore stale responses; each emit from the connection panel now
  // includes a `version` timestamp so the most recent result wins.
  if (
    existing
    && typeof version === 'number'
    && typeof existing.version === 'number'
    && existing.version > version
  ) {
    // an older query finished after a newer one; drop it
    return
  }

  const newTab = {
    key,
    title,
    result,
    error,
    explainResult: null,
    explainError: null,
    innerTab: 'result',
    version: version || Date.now(),
    context,
    loading: false,
    query: context?.action?.query || '',
    language: getMonacoLanguage(context?.conn?.driver_type),
  }

  if (error) {
    notification.error({ title: 'Query error', content: typeof error === 'string' ? error : String(error), duration: 6000 })
  }

  if (context && context.explain) {
    newTab.explainResult = result
    newTab.explainError = error
    newTab.innerTab = 'explain'
    if (existing) {
      newTab.result = existing.result
      newTab.error = existing.error
    }
    else {
      newTab.result = null
      newTab.error = null
    }
  }

  // if we already know the schema for this table, kick off a fetch so the
  // structure tab can display quickly when the user switches to it. we no
  // longer preselect the structure pane; result is always the starting point.
  const prefetchTable = extractTableName(newTab)
  if (prefetchTable) {
    if (getSchema(prefetchTable, newTab.context?.conn)) {
      // schema is cached, but don't switch tabs automatically
      console.debug('schema already cached for', prefetchTable, 'conn', newTab.context?.conn?.id)
    }
    else {
      // still initiate a fetch so the structure data will be available if the
      // user clicks the tab later.
      console.debug('openTab triggering fetchSchema for', prefetchTable, 'conn', newTab.context?.conn?.id)
      fetchSchema(prefetchTable, newTab.context?.conn).catch(err => console.error('fetchSchema failed', err))
    }
  }

  if (existing) {
    const idx = tabs.value.findIndex(t => t.key === key)
    if (idx !== -1) {
      tabs.value.splice(idx, 1, newTab)
    }
    else {
      tabs.value.push(newTab)
    }
  }
  else {
    tabs.value.push(newTab)
  }
  activeTabKey.value = key
}

function handleTabClose(closedKey) {
  tabs.value = tabs.value.filter(t => t.key !== closedKey)
  if (activeTabKey.value === closedKey) {
    activeTabKey.value = tabs.value.length ? tabs.value[0].key : ''
  }
  emit('tab-closed', closedKey)
}

function handleRefresh(tab) {
  if (!tab.context)
    return
  tab.loading = true
  tab.explainResult = null
  tab.explainError = null
  tab.innerTab = 'result'
  // Re-execute with the potentially modified query from the textbox.
  // If the existing context came from an earlier explain call it may
  // still contain an `explain` flag.  Removing that ensures the
  // subsequent execution is a normal query.
  const refreshContext = {
    ...tab.context,
    action: {
      ...tab.context.action,
      query: tab.query,
    },
  }
  if ('explain' in refreshContext) {
    delete refreshContext.explain
  }
  emit('refresh-tab', refreshContext)
}

defineExpose({ openTab })
</script>

<template>
  <div class="h-full overflow-hidden">
    <n-tabs
      v-model:value="activeTabKey"
      type="card"
      class="h-full"
      :tab-bar-style="{ position: 'sticky', top: 0, zIndex: 10 }"
      :pane-style="{ height: 'calc(100% - 41px)', overflow: 'hidden', padding: 0 }"
      @close="handleTabClose"
    >
      <n-tab-pane
        v-for="tab in tabs"
        :key="tab.key"
        :name="tab.key"
        :tab="tab.title || 'Untitled'"
        class="!p-0"
        closable
      >
        <template #default>
          <WelcomeTab v-if="tab.type === 'welcome'" />
          <div v-else-if="tab.result || tab.error" class="h-full overflow-hidden">
            <!-- Query Editor Area -->
            <div
              v-if="tab.context"
              class="border-b border-gray-200 bg-slate-50 relative h-48 pb-10"
            >
              <QueryEditor
                v-model="tab.query"
                :language="tab.language || 'sql'"
                :tab="tab"
                :connection="props.selectedConnection"
                @execute="handleRefresh(tab)"
              />
              <div class="absolute bottom-1.5 left-1.5 flex gap-2 z-10 pointer-events-none">
                <NButton
                  size="small"
                  type="primary"
                  :loading="tab.loading"
                  title="Execute (Ctrl+Enter)"
                  class="pointer-events-auto"
                  @click="handleRefresh(tab)"
                >
                  <template #icon>
                    <NIcon :size="12">
                      <Play />
                    </NIcon>
                  </template>
                  Execute
                </NButton>
                <NButton
                  v-if="supportsExplain(tab)"
                  size="small"
                  tertiary
                  :loading="tab.loading"
                  title="Explain query"
                  class="pointer-events-auto"
                  @click="handleExplain(tab)"
                >
                  <template #icon>
                    <NIcon :size="12">
                      <Analytics />
                    </NIcon>
                  </template>
                  Explain
                </NButton>
              </div>
            </div>

            <n-tabs
              v-model:value="tab.innerTab"
              type="line"
              animated
              :style="tab.context ? { height: 'calc(100% - 12rem)' } : { height: '100%' }"
              :pane-style="{ height: '100%', overflow: 'hidden', padding: 0 }"
              nav-wrapper-style="{paddingLeft:'0.5rem'}"
            >
              <template #prefix>
                &nbsp;
              </template>
              <n-tab-pane name="result" tab="Result">
                <template #default>
                  <ResultViewer v-if="tab.result" :result="tab.result" :schema="currentSchema" :connection="tab.context?.conn" :capabilities="tab.context?.capabilities ?? []" @mutated="handleRefresh(tab)" />
                  <pre
                    v-else-if="tab.error"
                    class="whitespace-pre-wrap p-4 text-red-600 bg-red-50 flex-1 overflow-auto font-mono text-sm"
                  >
{{ tab.error }}
                  </pre>
                  <div v-else class="text-gray-500 p-4">
                    No Results
                  </div>
                </template>
              </n-tab-pane>
              <n-tab-pane v-if="tab.explainResult || tab.explainError" name="explain" tab="Explain">
                <template #default>
                  <ResultViewer v-if="tab.explainResult" :result="tab.explainResult" class="pb-10" />
                  <pre
                    v-else-if="tab.explainError"
                    class="whitespace-pre-wrap p-4 text-red-600 bg-red-50 flex-1 overflow-auto font-mono text-sm"
                  >
{{ tab.explainError }}
                  </pre>
                </template>
              </n-tab-pane>
              <n-tab-pane v-if="currentSchema" name="structure" tab="Structure">
                <template #default>
                  <TableStructureViewer :schema="currentSchema" />
                </template>
              </n-tab-pane>
            </n-tabs>
          </div>
          <div v-else-if="tab.type !== 'welcome'" class="text-gray-500 p-4">
            No Results
          </div>
        </template>
      </n-tab-pane>
    </n-tabs>
  </div>
</template>

<style scoped>
:deep(.n-tab-pane) {
  padding: 0 !important;
}
</style>
