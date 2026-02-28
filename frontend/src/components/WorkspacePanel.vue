<script setup>
import { NButton, NIcon } from 'naive-ui'
import { ref, watch } from 'vue'
import QueryEditor from '@/components/QueryEditor.vue'
import ResultViewer from '@/components/ResultViewer.vue'
import { Analytics, Play, Refresh } from '@/lib/icons'

const props = defineProps({
  selectedConnection: { type: Object, default: null },
})
const emit = defineEmits(['tab-closed', 'active-connection-changed', 'refresh-tab'])

const tabs = ref([])
const activeTabKey = ref('')

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

watch(activeTabKey, (key) => {
  // tabKey format: conn.id + ":" + node.key — extract the connection ID
  const connId = key ? key.split(':')[0] : null
  emit('active-connection-changed', connId || null)
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
          <div v-if="tab.result || tab.error" class="h-full overflow-hidden">
            <!-- Query Editor Area -->
            <div
              v-if="tab.context"
              class="border-b border-gray-200 bg-slate-50 relative h-48 pb-10"
            >
              <QueryEditor
                v-model="tab.query"
                :language="tab.language || 'sql'"
                :context="tab.context"
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
                  <ResultViewer v-if="tab.result" :result="tab.result" />
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
            </n-tabs>
          </div>
          <div v-else class="text-gray-500 p-4">
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
