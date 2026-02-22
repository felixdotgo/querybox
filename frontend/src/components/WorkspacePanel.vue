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

    <!-- empty state when no tabs open -->
    <div
      v-if="tabs.length === 0"
      class="p-6 border border-dashed border-gray-200 rounded h-full flex items-center justify-center text-gray-500"
    >
      <div class="text-center">
        <div class="mb-2">
          Select a table (or other "select" action) from the tree to open a query tab.
        </div>
        <div v-if="selectedConnection" class="mt-4 text-left text-sm">
          <div><strong>Name:</strong> {{ selectedConnection.name }}</div>
          <div>
            <strong>Driver:</strong> {{ selectedConnection.driver_type }}
          </div>
        </div>
      </div>
    </div>
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

function openTab(title, result, error) {
  const key = `${Date.now()}-${Math.random().toString(36).slice(2)}`
  tabs.value.push({ key, title, result, error })
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
