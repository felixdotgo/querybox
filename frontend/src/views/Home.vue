<template>
  <div class="container-fluid p-0 flex flex-col h-screen bg-white">
    <!-- Header (application menu placeholder) -->
    <header
      ref="headerRef"
      :class="[
        'w-full border-b border-gray-200 pr-2 flex items-center flex-shrink-0',
        isMac ? 'pl-30 py-4 gap-2' : 'pl-2 gap-4 py-2',
      ]"
    >
      <n-button size="tiny" quaternary>File</n-button>
      <n-button size="tiny" quaternary>Edit</n-button>
      <n-button size="tiny" quaternary>View</n-button>
      <n-button size="tiny" quaternary>Help</n-button>
    </header>

    <!-- Content: two-column resizable layout -->
    <main class="flex-1 min-h-0">
      <TwoColumnLayout
        ref="layoutRef"
        :leftWidth="leftWidth"
        @dragstart="startDrag"
      >
        <template #left>
          <ConnectionsPanel
            @connection-selected="selectedConnection = $event"
            @query-result="openTab"
          />
        </template>

        <template #right>
          <WorkspacePanel
            ref="workspaceRef"
            :selectedConnection="selectedConnection"
          />
        </template>
      </TwoColumnLayout>
    </main>


    <!-- Footer resizer -->
    <div
      class="relative z-10 h-1 cursor-row-resize bg-gray-200 hover:bg-sky-500"
      @pointerdown="startFooterDrag"
    >
      <div class="w-10 h-0.5 bg-transparent mx-auto"></div>
    </div>

    <!-- Footer: collapsible log / debug area -->
    <footer class="w-full bg-white">
      <div class="flex items-center justify-between px-4 py-2">
        <div class="flex items-center gap-3">
          <button class="text-sm text-gray-600" @click="toggleFooter">
            <span v-if="footerCollapsed">Show logs</span>
            <span v-else>Hide logs</span>
          </button>
          <div class="text-xs opacity-60">Console / logs</div>
        </div>
        <div class="text-xs opacity-60">Status: ready</div>
      </div>

      <div
        v-show="!footerCollapsed"
        class="bg-gray-50 overflow-auto font-mono text-xs"
        :style="{ height: footerHeight + 'px' }"
      >
        <div v-if="logs.length === 0" class="p-3 text-gray-400 italic">
          Waiting for log output…
        </div>
        <div v-else class="p-2 space-y-0.5">
          <div
            v-for="(entry, i) in logs"
            :key="i"
            class="flex items-start gap-2 leading-5"
          >
            <span class="flex-shrink-0 text-gray-400">{{
              formatTime(entry.timestamp)
            }}</span>
            <span
              :class="[
                'flex-shrink-0 uppercase font-semibold w-10 text-center rounded px-1',
                entry.level === 'error'
                  ? 'bg-red-100 text-red-700'
                  : entry.level === 'warn'
                    ? 'bg-yellow-100 text-yellow-700'
                    : 'bg-green-100 text-green-700',
              ]"
              >{{ entry.level }}</span
            >
            <span class="text-gray-700 break-all">{{ entry.message }}</span>
          </div>
        </div>
        <div ref="logsEndRef" />
      </div>
    </footer>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from "vue"
import TwoColumnLayout from "@/components/TwoColumnLayout.vue"
import ConnectionsPanel from "@/components/ConnectionsPanel.vue"
import WorkspacePanel from "@/components/WorkspacePanel.vue"
import { Events } from "@wailsio/runtime"
// components required by footer etc (panels import their own viewers)
import {
  MinimiseMainWindow,
  ToggleFullScreenMainWindow,
  CloseMainWindow,
} from "@/bindings/github.com/felixdotgo/querybox/services/app"
import {
  createHorizontalResizer,
  createVerticalResizer,
} from "@/composables/useResize"

const isMac = navigator.userAgent.includes("Mac")


const headerRef = ref(null)
// reference to TwoColumnLayout component instance; exposes inner containerRef
const layoutRef = ref(null)
const leftWidth = ref(0)

// convenience computed pointing at actual DOM element
const containerRef = computed(() => layoutRef.value?.containerRef)
const selectedConnection = ref(null)

const footerCollapsed = ref(false)
const footerHeight = ref(176)

// log entries streamed from the Go backend via the app:log event
const logs = ref([])
const logsEndRef = ref(null)
let offAppLog = null

// reusable resizers (horizontal + vertical)
const horizontalResizer = createHorizontalResizer({
  containerRef,
  sizeRef: leftWidth,
  min: 400,
  minOther: 200,
})

const verticalResizer = createVerticalResizer({
  sizeRef: footerHeight,
  min: 80,
  getMax: () => {
    const headerH = headerRef.value?.getBoundingClientRect().height ?? 0
    const minMainHeight = 120
    return Math.max(80, window.innerHeight - headerH - minMainHeight)
  },
})


// workspace reference used to add tabs via exposed method
const workspaceRef = ref(null)

function openTab(title, result, error, tabKey, version) {
  // pass the optional version through so workspace can ignore stale data
  workspaceRef.value?.openTab(title, result, error, tabKey, version)
}


function startDrag(e) {
  horizontalResizer.start(e)
}

function startFooterDrag(e) {
  if (footerCollapsed.value) footerCollapsed.value = false
  verticalResizer.start(e)
}

// (removed) onPointerMove — using dedicated listeners for horizontal and vertical drags

// resizing cleanup handled by composables

function toggleFooter() {
  footerCollapsed.value = !footerCollapsed.value
  if (
    !footerCollapsed.value &&
    (!footerHeight.value || footerHeight.value < 80)
  ) {
    footerHeight.value = 176
  }
}

// Format an RFC3339 timestamp to a short HH:MM:SS.mmm string.
// Go's RFC3339Nano uses 9-digit nanoseconds; JS Date only parses up to 3,
// so we truncate fractional seconds to 3 digits before constructing the Date.
function formatTime(ts) {
  try {
    const normalized = ts.replace(/(\.\d{3})\d+/, "$1")
    const d = new Date(normalized)
    if (isNaN(d.getTime())) return ts
    const hh = String(d.getHours()).padStart(2, "0")
    const mm = String(d.getMinutes()).padStart(2, "0")
    const ss = String(d.getSeconds()).padStart(2, "0")
    const ms = String(d.getMilliseconds()).padStart(3, "0")
    return `${hh}:${mm}:${ss}.${ms}`
  } catch {
    return ts
  }
}

const resizeHandler = () => {
  horizontalResizer.clamp()
  verticalResizer.clamp()
}

onMounted(() => {
  // initialize left column to 1/5 of container but at least 400px
  const rect = containerRef.value?.getBoundingClientRect()
  const cw = rect?.width ?? window.innerWidth
  leftWidth.value = Math.max(Math.floor(cw * 0.2), 400)

  // default footer height
  footerHeight.value = 176

  // Stream Go backend log events into the Logs tab.
  // Events.On callback receives a WailsEvent; the actual payload is in .data.
  offAppLog = Events.On("app:log", (event) => {
    const entry = event?.data ?? event
    if (!entry) return
    logs.value.push(entry)
    if (!footerCollapsed.value) {
      nextTick(() => logsEndRef.value?.scrollIntoView({ behavior: "smooth" }))
    }
  })

  // ensure initial sizes are within computed bounds
  horizontalResizer.clamp()
  verticalResizer.clamp()

  window.addEventListener("resize", resizeHandler)
})

onUnmounted(() => {
  window.removeEventListener("resize", resizeHandler)
  horizontalResizer.destroy()
  verticalResizer.destroy()
  if (offAppLog) offAppLog()
})
</script>
