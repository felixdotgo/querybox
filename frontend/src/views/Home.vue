<script setup>
import { Events } from '@wailsio/runtime'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ConnectionsPanel } from '@/components/connections'
import { AppMenuBar, LogsPanel, SafeZone } from '@/components/layout'
import { WorkspacePanel } from '@/components/workspace'
import { ChevronDown, Terminal } from '@/lib/icons'

const isMac = navigator.userAgent.includes('Mac')
const menuBarRef = ref(null)

const selectedConnection = ref(null)
const activeConnectionId = ref(null)

// footer state
const footerCollapsedFlag = ref(true) // pixels of footer when expanded
// reactive collapsed flag derived from size
const footerCollapsed = computed(() => footerCollapsedFlag.value === true)

const footerDefaultSize = computed(() => {
  if (footerCollapsed.value) {
    return 1
  }
  return 0.9
})

// log entries streamed from the Go backend via the app:log event
const logs = ref([])
let offAppLog = null

function clearLogs() {
  logs.value = []
}

function toggleFooter() {
  footerCollapsedFlag.value = !footerCollapsedFlag.value
}

// workspace reference used to add tabs via exposed method
const workspaceRef = ref(null)
const connectionsRef = ref(null)

function openTab(title, result, error, tabKey, version, context) {
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

onMounted(() => {
  offAppLog = Events.On('app:log', (event) => {
    const entry = event?.data ?? event
    if (!entry)
      return
    logs.value.push(entry)
  })

  Events.On('menu:logs-toggled', () => toggleFooter())
})

onUnmounted(() => {
  if (offAppLog)
    offAppLog()
})
</script>

<template>
  <div class="relative flex flex-col h-screen bg-white">
    <AppMenuBar v-if="!isMac" ref="menuBarRef" @toggle-logs="toggleFooter" />

    <main class="flex-1 min-h-0 overflow-hidden">
      <n-split
        direction="vertical"
        :min="0.1"
        :max="!footerCollapsed ? 0.9 : 1"
        :default-size="footerDefaultSize"
        :watch-props="['defaultSize']"
      >
        <template #1>
          <n-split
            direction="horizontal"
            default-size="0.2"
            :pane-1-style="{ minWidth: '250px' }"
            :pane-2-style="{ minWidth: '200px' }"
          >
            <template #1>
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
            <template #2>
              <WorkspacePanel
                ref="workspaceRef"
                :selected-connection="selectedConnection"
                @active-connection-changed="activeConnectionId = $event"
                @refresh-tab="handleRefreshTab"
              />
            </template>
          </n-split>
        </template>
        <template #2>
          <div v-show="!footerCollapsed" class="w-full overflow-hidden h-full">
            <LogsPanel :logs="logs" @clear="clearLogs" />
          </div>
        </template>
      </n-split>
    </main>
  </div>
</template>
