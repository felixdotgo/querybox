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
      <div ref="containerRef" class="flex w-full h-full overflow-hidden">
        <!-- Left column: toolbar + tree -->
        <div
          class="flex-shrink-0"
          :style="{ width: leftWidth + 'px', minWidth: '400px' }"
        >
          <div class="p-3 h-full flex flex-col gap-3">
            <!-- small toolbar -->
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-2">
                <span class="text-lg font-semibold m-0">Databases</span>
              </div>
              <div class="flex items-center gap-2">
                <n-button
                  size="small"
                  secondary
                  title="New connection"
                  @click="openConnections"
                  >+</n-button
                >
              </div>
            </div>

            <n-input
              v-model:value="filter"
              size="small"
              placeholder="Filter connections"
            />

            <div class="flex-1 overflow-auto mt-2 px-1 min-h-0">
              <n-tree
                :data="filteredTreeData"
                v-model:expanded-keys="expandedKeys"
                :default-expanded-keys="defaultExpandedKeys"
                node-key="key"
                block-node
                :show-selector="false"
                :node-props="getNodeProps"
                :render-label="renderLabel"
                @update:selected-keys="handleSelect"
              />
              <div
                v-if="connections.length === 0"
                class="py-6 text-center opacity-70"
              >
                No connections configured
              </div>
            </div>
          </div>
        </div>

        <!-- Drag divider -->
        <div
          role="separator"
          aria-orientation="vertical"
          class="w-1 cursor-col-resize bg-gray-200 hover:bg-sky-500"
          @pointerdown="startDrag"
        >
          <div class="w-0.5 h-full bg-transparent mx-auto"></div>
        </div>

        <!-- Right column: placeholder (functional details to be decided later) -->
        <div class="flex-1 p-4 min-h-0 overflow-auto">
          <span class="text-lg font-semibold mb-2">Workspace</span>
          <div
            class="p-6 border border-dashed border-gray-200 rounded h-full flex items-center justify-center text-gray-500"
          >
            <div class="text-center">
              <div class="mb-2">
                Right pane placeholder — functionality to be decided
              </div>
              <div v-if="selectedConnection" class="mt-4 text-left text-sm">
                <div><strong>Name:</strong> {{ selectedConnection.name }}</div>
                <div>
                  <strong>Driver:</strong> {{ selectedConnection.driver_type }}
                </div>
                <div class="opacity-70 text-xs mt-2">
                  (Selecting a connection will later open editors / query tabs)
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>

    <!-- Delete confirmation overlay -->
    <div
      v-if="deleteModal.visible"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
      @click.self="deleteModal.visible = false"
    >
      <div class="bg-white rounded-lg shadow-xl p-6 w-80">
        <div class="text-base font-semibold mb-2">Delete connection</div>
        <div class="text-sm text-gray-600 mb-5">
          Delete <strong>{{ deleteModal.conn?.name }}</strong
          >? This cannot be undone.
        </div>
        <div class="flex justify-end gap-2">
          <button
            class="px-4 py-1.5 text-sm rounded border border-gray-300 hover:bg-gray-50"
            @click="deleteModal.visible = false"
          >
            Cancel
          </button>
          <button
            class="px-4 py-1.5 text-sm rounded bg-red-600 text-white hover:bg-red-700"
            @click="confirmDelete"
          >
            Delete
          </button>
        </div>
      </div>
    </div>

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
import { ref, computed, h, watch, onMounted, onUnmounted, nextTick } from "vue"
import { Events } from "@wailsio/runtime"
import { useRouter } from "vue-router"
import {
  ListConnections,
  GetCredential,
  DeleteConnection,
} from "@/bindings/github.com/felixdotgo/querybox/services/connectionservice"
// @ts-ignore: may be generated after adding new methods
import {
  GetConnectionTree,
  ExecTreeAction,
  ExecPlugin,
} from "@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager"
import {
  ShowConnectionsWindow,
  MinimiseMainWindow,
  ToggleFullScreenMainWindow,
  CloseMainWindow,
} from "@/bindings/github.com/felixdotgo/querybox/services/app"
import {
  createHorizontalResizer,
  createVerticalResizer,
} from "@/composables/useResize"

const isMac = navigator.userAgent.includes("Mac")

const router = useRouter()
async function openConnections() {
  try {
    await ShowConnectionsWindow()
    return
  } catch (err) {
    router.push("/connections")
  }
}

const headerRef = ref(null)
const containerRef = ref(null)
const leftWidth = ref(0)
const connections = ref([])
const filter = ref("")
const selectedConnection = ref(null)
// map from connection ID to array of tree nodes returned by plugin
const connectionTrees = ref({})

// delete confirmation modal state
const deleteModal = ref({ visible: false, conn: null })
// keys which should be expanded in the tree (drivers + per-connection IDs)
const expandedKeys = ref([])

const footerCollapsed = ref(true)
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

const defaultExpandedKeys = computed(() => {
  return treeData.value.map((g) => g.key)
})

const treeData = computed(() => {
  const groups = {}
  for (const c of connections.value || []) {
    const key = c.driver_type || "unknown"
    if (!groups[key]) groups[key] = []
    groups[key].push(c)
  }
  return Object.entries(groups).map(([driver, conns]) => ({
    key: `driver:${driver}`,
    label: `${driver} (${conns.length})`,
    children: conns.map((cc) => {
      const extra = connectionTrees.value[cc.id] || []
      return { key: cc.id, label: cc.name, children: extra }
    }),
  }))
})

const filteredTreeData = computed(() => {
  const q = (filter.value || "").toLowerCase().trim()
  if (!q) return treeData.value
  return treeData.value
    .map((g) => ({
      ...g,
      children: g.children.filter((ch) =>
        (ch.label || "").toLowerCase().includes(q),
      ),
    }))
    .filter(
      (g) => g.children.length > 0 || (g.label || "").toLowerCase().includes(q),
    )
})

async function loadConnections() {
  try {
    connections.value = (await ListConnections()) || []
    // clear any cached trees so that double-clicks on the refreshed list
    // will trigger a new fetch. this prevents stale state from previous
    // connections interfering with the UI.
    connectionTrees.value = {}
  } catch (err) {
    console.error("ListConnections", err)
    connections.value = []
    connectionTrees.value = {}
  }
}

// NaiveUI tree fires @update:selected-keys with an array of selected keys
function handleSelect(keys, options, meta) {
  // extract the single key that was just acted on
  const key = meta?.node?.key ?? (Array.isArray(keys) ? keys[0] : keys)
  if (key == null) return
  console.log("handleSelect", { key, meta, options })

  // leaf keys are connection IDs
  const conn = connections.value.find((c) => c.id === key)
  if (conn) {
    console.log("selected connection", conn)
    selectedConnection.value = conn
    // single-click: fetch the tree if not already loaded
    fetchTreeFor(conn)
    return
  }

  // if not a top-level connection, it may be a plugin-provided node with
  // actions. `selectedConnection` holds the parent connection in that case.
  const parentConn = selectedConnection.value
  const node = meta?.node
  if (parentConn && node && node.actions && node.actions.length > 0) {
    // pick first action for now
    const act = node.actions[0]
    runTreeAction(parentConn, act)
  }
}

// Returns per-node DOM props for double-click on connections.
// shared helper invoked when the user double-clicks a connection node
function handleConnectionDblclick(conn) {
  if (!conn) return
  console.log("dblclick connection", conn.id)
  selectedConnection.value = conn

  // clear cached tree so we always fetch fresh data
  const copy = { ...connectionTrees.value }
  delete copy[conn.id]
  connectionTrees.value = copy

  checkConnection(conn)
  fetchTreeFor(conn)
}

function getNodeProps(node) {
  const conn = connections.value.find((c) => c.id === node.key)
  if (!conn) return {}
  return {
    // Vue's vnode props for events expect camelCase keys like `onDblclick`.
    // using lowercase `ondblclick` did not always attach listener after
    // the connections list was refreshed. switching to the proper key and
    // mutating the tree state immutably ensures the handler is always
    // registered and reactive updates fire.
    onDblclick(e) {
      e.stopPropagation()
      handleConnectionDblclick(conn)
    },
  }
}

// Renders each tree node label; connection nodes get an inline hover delete button.
function renderLabel({ option }) {
  const conn = connections.value.find((c) => c.id === option.key)
  if (!conn) {
    // non-connection node (driver group or plugin child) — plain label
    return option.label
  }
  return h(
    "div",
    {
      class: "flex items-center justify-between w-full group/conn pr-1",
      onDblclick: (e) => {
        e.stopPropagation()
        handleConnectionDblclick(conn)
      },
    },
    [
      h("span", { class: "truncate" }, option.label),
      h(
        "button",
        {
          class:
            "opacity-0 group-hover/conn:opacity-100 ml-2 flex-shrink-0 text-gray-400 hover:text-red-500 transition-opacity leading-none",
          title: "Delete connection",
          onClick(e) {
            e.stopPropagation()
            deleteModal.value = { visible: true, conn }
          },
        },
        "×",
      ),
    ],
  )
}

async function confirmDelete() {
  const conn = deleteModal.value.conn
  if (!conn) return
  try {
    await DeleteConnection(conn.id)
    // remove from local state
    connections.value = connections.value.filter((c) => c.id !== conn.id)
    delete connectionTrees.value[conn.id]
    if (selectedConnection.value?.id === conn.id)
      selectedConnection.value = null
    // remove from expanded keys
    expandedKeys.value = expandedKeys.value.filter((k) => k !== conn.id)
  } catch (err) {
    console.error("DeleteConnection", err)
  } finally {
    deleteModal.value = { visible: false, conn: null }
  }
}

async function runTreeAction(conn, action) {
  console.log("runTreeAction", conn.id, action)
  try {
    const cred = await GetCredential(conn.id)
    console.log("credential fetched", cred)
    const params = {}
    if (cred) params.credential_blob = cred
    const res = await ExecTreeAction(conn.driver_type, params, action.query)
    console.log("action result", res)
    // TODO: render result in workspace
  } catch (err) {
    console.error("ExecTreeAction", conn.id, err)
  }
}

async function checkConnection(conn) {
  console.log("checkConnection start", conn.id)
  // attempt a simple query to verify connection validity; result is ignored
  try {
    const cred = await GetCredential(conn.id)
    console.log("credential for check", cred)
    const params = {}
    if (cred) params.credential_blob = cred
    const res = await ExecPlugin(conn.driver_type, params, "SELECT 1")
    console.log("connection check result", conn.id, res)
  } catch (err) {
    console.error("connection check", conn.id, err)
  }
}

async function fetchTreeFor(conn) {
  console.log("fetchTreeFor", conn.id)
  if (connectionTrees.value[conn.id]) {
    console.log("tree already cached for", conn.id)
    return
  }
  try {
    const cred = await GetCredential(conn.id)
    console.log("credential for tree", cred)
    const params = {}
    if (cred) params.credential_blob = cred
    const resp = await GetConnectionTree(conn.driver_type, params)
    console.log("got tree for", conn.id, resp)
    // assign immutably to guarantee reactivity
    connectionTrees.value = {
      ...connectionTrees.value,
      [conn.id]: resp.nodes || [],
    }
    // when nodes arrive, make sure the connection node is expanded so user
    // sees the children without needing to click the arrow manually
    if (!expandedKeys.value.includes(conn.id)) {
      expandedKeys.value.push(conn.id)
    }
  } catch (err) {
    console.error("GetConnectionTree", conn.id, err)
    connectionTrees.value = { ...connectionTrees.value, [conn.id]: [] }
  }
}

function startDrag(e) {
  horizontalResizer.start(e)
}

// keep tree expanded when a connection is selected manually
watch(selectedConnection, (conn) => {
  if (conn && !expandedKeys.value.includes(conn.id)) {
    expandedKeys.value.push(conn.id)
  }
})

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

let offConnectionSaved = null

onMounted(async () => {
  // initialize left column to 1/5 of container but at least 400px
  const rect = containerRef.value?.getBoundingClientRect()
  const cw = rect?.width ?? window.innerWidth
  leftWidth.value = Math.max(Math.floor(cw * 0.2), 400)

  // default footer height
  footerHeight.value = 176

  await loadConnections()

  // Reload connections list whenever a new connection is saved from the
  // Connections window.
  offConnectionSaved = Events.On("connection:saved", () => {
    loadConnections()
  })

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

  // make sure driver nodes start expanded when data arrives
  expandedKeys.value = defaultExpandedKeys.value

  // ensure initial sizes are within computed bounds
  horizontalResizer.clamp()
  verticalResizer.clamp()

  window.addEventListener("resize", resizeHandler)
})

onUnmounted(() => {
  window.removeEventListener("resize", resizeHandler)
  horizontalResizer.destroy()
  verticalResizer.destroy()
  if (offConnectionSaved) offConnectionSaved()
  if (offAppLog) offAppLog()
})
</script>
