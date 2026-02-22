<template>
  <div class="flex flex-col h-full">
    <span class="text-lg font-semibold mb-2">Workspace</span>
    <n-tabs
      type="card"
      v-model:value="activeTabKey"
      @close="handleTabClose"
      class="mb-4"
    >
      <n-tab-pane
        v-for="tab in tabs"
        :key="tab.key"
        :name="tab.key"
        :title="tab.title || 'Untitled'"
        closable
      >

        <template #default>
          <ResultViewer v-if="tab.result" :result="tab.result" />
          <pre v-else-if="tab.error" class="whitespace-pre-wrap">
{{ tab.error }}
          </pre>
          <div v-else class="text-gray-500">
            No data to display
          </div>
        </template>
      </n-tab-pane>
    </n-tabs>
  </div>
</template>

<script setup>
import { ref } from "vue"
import ResultViewer from "@/components/ResultViewer.vue"

const props = defineProps({
  selectedConnection: { type: Object, default: null },
})
const emit = defineEmits(["tab-closed"])

const tabs = ref([])
const activeTabKey = ref("")

function openTab(title, result, error, tabKey, version) {
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

  const newTab = { key, title, result, error, version: version || Date.now() }

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

defineExpose({ openTab })
</script>
