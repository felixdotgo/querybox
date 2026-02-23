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
      class="relative z-10 h-1 cursor-row-resize bg-gray-200 hover:bg-blue-400 transition-colors"
      @pointerdown="startFooterDrag"
    >
      <div class="w-10 h-0.5 bg-transparent mx-auto"></div>
    </div>

    <!-- Footer: collapsible log / debug area -->
    <footer class="w-full border-t border-gray-200">
      <!-- Collapse toggle bar -->
      <div class="flex items-center justify-between px-3 py-1 border-b border-gray-200 bg-gray-50">
        <button
          class="flex items-center gap-1.5 text-[11px] text-gray-500 hover:text-gray-800 transition-colors font-mono"
          @click="toggleFooter"
        >
          <svg
            :class="['w-3 h-3 transition-transform', footerCollapsed ? '-rotate-90' : '']"
            viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"
          >
            <path d="M4 6l4 4 4-4"/>
          </svg>
          Logs
        </button>
        <span class="text-[10px] text-gray-400">{{ logs.length }} events</span>
      </div>

      <div
        v-show="!footerCollapsed"
        :style="{ height: footerHeight + 'px' }"
        class="overflow-hidden"
      >
        <LogsPanel :logs="logs" @clear="clearLogs" />
      </div>
    </footer>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from "vue"
import TwoColumnLayout from "@/components/TwoColumnLayout.vue"
import ConnectionsPanel from "@/components/ConnectionsPanel.vue"
import WorkspacePanel from "@/components/WorkspacePanel.vue"
import LogsPanel from "@/components/LogsPanel.vue"
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
let offAppLog = null

function clearLogs() {
  logs.value = []
}

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
