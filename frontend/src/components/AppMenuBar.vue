<template>
  <header
    ref="el"
    :class="[
      'w-full border-b border-gray-200 pr-2 flex items-center flex-shrink-0',
      isMac ? 'pl-30 py-0.5 gap-0' : 'pl-1 gap-0 py-0.5',
    ]"
  >
    <n-dropdown trigger="click" :options="fileMenu" @select="handleSelect">
      <n-button size="tiny" quaternary class="rounded-none px-3">File</n-button>
    </n-dropdown>
    <n-dropdown trigger="click" :options="viewMenu" @select="handleSelect">
      <n-button size="tiny" quaternary class="rounded-none px-3">View</n-button>
    </n-dropdown>

    <!-- Slot for window-specific quick actions -->
    <template v-if="$slots.actions">
      <n-divider vertical class="mx-2" />
      <slot name="actions" />
    </template>
  </header>
</template>

<script setup>
import { ref } from "vue"
import {
  ShowConnectionsWindow,
  CloseMainWindow,
  ToggleFullScreenMainWindow,
} from "@/bindings/github.com/felixdotgo/querybox/services/app"

const emit = defineEmits(["toggle-logs"])

const isMac = navigator.userAgent.includes("Mac")

// Exposed so parents can measure header height for layout calculations.
const el = ref(null)
defineExpose({ el })

const fileMenu = [
  { label: "New Connection", key: "new-connection" },
  { type: "divider", key: "d1" },
  { label: "Quit", key: "quit" },
]

const viewMenu = [
  { label: "Toggle Fullscreen", key: "toggle-fullscreen" },
  { label: "Toggle Logs", key: "toggle-logs" },
]

function handleSelect(key) {
  switch (key) {
    case "new-connection":    ShowConnectionsWindow(); break
    case "quit":              CloseMainWindow(); break
    case "toggle-fullscreen": ToggleFullScreenMainWindow(); break
    case "toggle-logs":       emit("toggle-logs"); break
  }
}
</script>
