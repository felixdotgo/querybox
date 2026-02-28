<script setup>
import { Events } from '@wailsio/runtime'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import AppMenuBar from '@/components/AppMenuBar.vue'
import ConnectionsPanel from '@/components/ConnectionsPanel.vue'
import LogsPanel from '@/components/LogsPanel.vue'
import SafeZone from '@/components/SafeZone.vue'
import TwoColumnLayout from '@/components/TwoColumnLayout.vue'
import WorkspacePanel from '@/components/WorkspacePanel.vue'
import {
  createHorizontalResizer,
  createVerticalResizer,
} from '@/composables/useResize'
import { ChevronDown, Terminal } from '@/lib/icons'

const isMac = navigator.userAgent.includes('Mac')
const menuBarRef = ref(null)
// reference to TwoColumnLayout component instance; exposes inner containerRef
const layoutRef = ref(null)
const leftWidth = ref(0)

// convenience computed pointing at actual DOM element
const containerRef = computed(() => layoutRef.value?.containerRef)
const selectedConnection = ref(null)
const activeConnectionId = ref(null)

const footerCollapsed = ref(true)
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
  min: 250,
  minOther: 200,
})

const verticalResizer = createVerticalResizer({
  sizeRef: footerHeight,
  min: 80,
  getMax: () => {
    const headerH = menuBarRef.value?.el?.getBoundingClientRect().height ?? 0
    const minMainHeight = 120
    return Math.max(80, window.innerHeight - headerH - minMainHeight)
  },
})

// workspace reference used to add tabs via exposed method
const workspaceRef = ref(null)
const connectionsRef = ref(null)

function openTab(title, result, error, tabKey, version, context) {
  // pass the optional version through so workspace can ignore stale data
  workspaceRef.value?.openTab(title, result, error, tabKey, version, context)
}

function handleRefreshTab(context) {
  if (!context || !context.conn || !context.action)
    return
  const extras = {}
  if (context.explain) {
    extras.explain = true
  }
  connectionsRef.value?.runTreeAction(context.conn, context.action, context.node, extras)
}

function startDrag(e) {
  horizontalResizer.start(e)
}

function startFooterDrag(e) {
  if (footerCollapsed.value)
    footerCollapsed.value = false
  verticalResizer.start(e)
}

// (removed) onPointerMove — using dedicated listeners for horizontal and vertical drags

// resizing cleanup handled by composables

function toggleFooter() {
  footerCollapsed.value = !footerCollapsed.value
  if (
    !footerCollapsed.value
    && (!footerHeight.value || footerHeight.value < 80)
  ) {
    footerHeight.value = 176
  }
}

function resizeHandler() {
  horizontalResizer.clamp()
  verticalResizer.clamp()
}

onMounted(() => {
  // initialize left column to 1/5 of container but at least 250px
  const rect = containerRef.value?.getBoundingClientRect()
  const cw = rect?.width ?? window.innerWidth
  leftWidth.value = Math.max(Math.floor(cw * 0.2), 250)

  // default footer height
  footerHeight.value = 176

  // Stream Go backend log events into the Logs tab.
  // Events.On callback receives a WailsEvent; the actual payload is in .data.
  offAppLog = Events.On('app:log', (event) => {
    const entry = event?.data ?? event
    if (!entry)
      return
    logs.value.push(entry)
  })

  Events.On('menu:logs-toggled', () => toggleFooter())

  // ensure initial sizes are within computed bounds
  horizontalResizer.clamp()
  verticalResizer.clamp()

  window.addEventListener('resize', resizeHandler)
})

onUnmounted(() => {
  window.removeEventListener('resize', resizeHandler)
  horizontalResizer.destroy()
  verticalResizer.destroy()
  if (offAppLog)
    offAppLog()
})
</script>

<template>
  <div class="flex flex-col h-screen bg-white">
    <AppMenuBar v-if="!isMac" ref="menuBarRef" @toggle-logs="toggleFooter" />

    <!-- Content: two-column resizable layout -->
    <main class="flex-1 min-h-0 overflow-hidden">
      <TwoColumnLayout
        ref="layoutRef"
        :left-width="leftWidth"
        @dragstart="startDrag"
      >
        <template #left>
          <div class="bg-slate-50 h-full">
            <SafeZone />
            <ConnectionsPanel
              ref="connectionsRef"
              :active-connection-id="activeConnectionId"
              @connection-selected="selectedConnection = $event"
              @query-result="openTab"
            />
          </div>
        </template>

        <template #right>
          <WorkspacePanel
            ref="workspaceRef"
            :selected-connection="selectedConnection"
            @active-connection-changed="activeConnectionId = $event"
            @refresh-tab="handleRefreshTab"
          />
        </template>
      </TwoColumnLayout>
    </main>

    <!-- Footer resizer -->
    <div
      class="relative z-10 h-1 cursor-row-resize bg-gray-200 hover:bg-blue-400 transition-colors"
      @pointerdown="startFooterDrag"
    >
      <div class="w-10 h-0.5 bg-transparent mx-auto" />
    </div>

    <!-- Footer: collapsible log / debug area -->
    <footer class="w-full border-t border-gray-200">
      <!-- Collapse toggle bar -->
      <div class="flex items-center justify-between px-3 py-1 border-b border-gray-200 bg-gray-50">
        <button
          class="flex items-center gap-1.5 text-[11px] text-gray-500 hover:text-gray-800 transition-colors font-mono"
          @click="toggleFooter"
        >
          <n-icon
            :size="12"
            class="transition-transform" :class="[footerCollapsed ? '-rotate-90' : '']"
          >
            <ChevronDown />
          </n-icon>
          <n-icon :size="12">
            <Terminal />
          </n-icon>
          Logs
        </button>
        <span class="text-[10px] text-gray-400">{{ logs.length }} operations</span>
      </div>

      <div
        v-show="!footerCollapsed"
        :style="{ height: `${footerHeight}px` }"
        class="overflow-hidden"
      >
        <LogsPanel :logs="logs" @clear="clearLogs" />
      </div>
    </footer>
  </div>
</template>
