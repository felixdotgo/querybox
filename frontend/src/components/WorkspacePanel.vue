<template>
  <div class="flex flex-col h-full overflow-hidden">
    <n-tabs
      type="card"
      v-model:value="activeTabKey"
      @close="handleTabClose"
      class="flex flex-col h-full"
      :tab-bar-style="{ position: 'sticky', top: 0, zIndex: 10, flexShrink: 0 }"
      :pane-style="{ display: 'flex', flexDirection: 'column', overflow: 'hidden', flex: '1 1 0', minHeight: 0, padding: 0 }"
    >
      <n-tab-pane
        v-for="tab in tabs"
        :key="tab.key"
        :name="tab.key"
        :tab="tab.title || 'Untitled'"
        closable
      >

        <template #default>
          <div v-if="tab.result || tab.error" class="flex flex-col h-full overflow-hidden">
            <!-- Query Editor Area -->
            <div
              v-if="tab.context"
              class="p-2 border-b border-gray-100 flex flex-col gap-2 bg-slate-50 shrink-0"
            >
              <n-input
                v-model:value="tab.query"
                type="textarea"
                :autosize="{ minRows: 2, maxRows: 10 }"
                placeholder="Type query..."
                size="small"
                class="font-mono text-[11px]"
                @keypress.enter.prevent.ctrl="handleRefresh(tab)"
              />
              <div class="flex">
                <n-button
                  size="small"
                  type="primary"
                  :loading="tab.loading"
                  @click="handleRefresh(tab)"
                  title="Execute (Ctrl+Enter)"
                >
                  <template #icon>
                    <n-icon :size="12"><RefreshOutline /></n-icon>
                  </template>
                  Execute
                </n-button>
              </div>
            </div>

            <ResultViewer v-if="tab.result" :result="tab.result" class="flex-1" />
            <pre
              v-else-if="tab.error"
              class="whitespace-pre-wrap p-4 text-red-600 bg-red-50 flex-1 overflow-auto font-mono text-sm"
            >
{{ tab.error }}
            </pre>
          </div>
          <div v-else class="text-gray-500 p-4">
            No Results
          </div>
        </template>
      </n-tab-pane>
    </n-tabs>
  </div>
</template>

<script setup>
import { ref, watch } from "vue"
import { NButton, NIcon, NInput } from "naive-ui"
import { RefreshOutline } from "@/lib/icons"
import ResultViewer from "@/components/ResultViewer.vue"

const props = defineProps({
  selectedConnection: { type: Object, default: null },
})
const emit = defineEmits(["tab-closed", "active-connection-changed", "refresh-tab"])

const tabs = ref([])
const activeTabKey = ref("")

watch(activeTabKey, (key) => {
  // tabKey format: conn.id + ":" + node.key — extract the connection ID
  const connId = key ? key.split(":")[0] : null
  emit("active-connection-changed", connId || null)
})

function openTab(title, result, error, tabKey, version, context) {
  // sanitize human title just in case it still contains a prefix
  const sanitize = (t) => (t ? t.split(":").pop() : t)
  title = sanitize(title)

  // `tabKey` is a stable identifier used internally to avoid opening
  // duplicate tabs. the human‑readable title shown on the tab is always
  // supplied separately (usually the node.key such as "db.table").
  // only when `tabKey` is absent do we fall back to the title, and as a
  // last resort we generate a random id.
  let key
  if (tabKey) {
    key = tabKey
  } else if (title) {
    key = title
  } else {
    key = `${Date.now()}-${Math.random().toString(36).slice(2)}`
  }

  // migration support for older tabs that used the title as the key.
  let existing = tabs.value.find((t) => t.key === key)
  if (!existing && tabKey && title) {
    const alt = tabs.value.find((t) => t.key === title)
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
    existing &&
    typeof version === "number" &&
    typeof existing.version === "number" &&
    existing.version > version
  ) {
    // an older query finished after a newer one; drop it
    return
  }

  const newTab = {
    key,
    title,
    result,
    error,
    version: version || Date.now(),
    context,
    loading: false,
    query: context?.action?.query || "",
  }

  if (existing) {
    const idx = tabs.value.findIndex((t) => t.key === key)
    if (idx !== -1) {
      tabs.value.splice(idx, 1, newTab)
    } else {
      tabs.value.push(newTab)
    }
  } else {
    tabs.value.push(newTab)
  }
  activeTabKey.value = key
}

function handleTabClose(closedKey) {
  tabs.value = tabs.value.filter((t) => t.key !== closedKey)
  if (activeTabKey.value === closedKey) {
    activeTabKey.value = tabs.value.length ? tabs.value[0].key : ""
  }
  emit("tab-closed", closedKey)
}

function handleRefresh(tab) {
  if (!tab.context) return
  tab.loading = true
  // Re-execute with the potentially modified query from the textbox
  const refreshContext = {
    ...tab.context,
    action: {
      ...tab.context.action,
      query: tab.query
    }
  }
  emit("refresh-tab", refreshContext)
}

defineExpose({ openTab })
</script>
