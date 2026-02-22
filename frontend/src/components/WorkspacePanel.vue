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
