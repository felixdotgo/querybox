<template>
  <div class="p-3 h-full flex flex-col gap-3 bg-slate-50">
    <!-- small toolbar -->
    <div class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <span class="text-lg font-semibold m-0">Connections</span>
      </div>
      <div class="flex items-center gap-2">
        <n-button
          size="small"
          secondary
          title="New connection"
          @click="openConnections"
        >
          <template #icon>
            <n-icon><AddCircleOutline /></n-icon>
          </template>
        </n-button>
      </div>
    </div>

    <n-input
      v-model:value="filter"
      size="small"
      placeholder="Filter"
    />

    <div class="flex-1 overflow-auto mt-2 px-1 min-h-0">
      <n-tree
        :data="filteredTreeData"
        v-model:expanded-keys="expandedKeys"
        :default-expanded-keys="defaultExpandedKeys"
        :node-key="node => node.key"
        block-node
        :show-selector="false"
        :node-props="getNodeProps"
        :render-label="renderLabel"
        :render-prefix="renderPrefix"
        @update:selected-keys="handleSelect"
      />
      <div
        v-if="connections.length === 0"
        class="py-6 text-center opacity-70"
      >
        No connections yet
      </div>
    </div>

    <!-- delete confirmation dialog -->
    <n-modal
      v-model:show="deleteModal.visible"
      preset="dialog"
      type="error"
      title="Remove connection"
      :content="`Remove &quot;${deleteModal.conn?.name}&quot;? This cannot be undone.`"
      positive-text="Remove"
      negative-text="Cancel"
      @positive-click="confirmDelete"
      @negative-click="deleteModal.visible = false"
    />
  </div>
</template>

<script setup>
import { ref, computed, h, watch, onUnmounted, defineEmits } from "vue"
import { NIcon } from "naive-ui"
import { Events } from "@wailsio/runtime"
import { useRouter } from "vue-router"
import ConnectionNodeLabel from "@/components/ConnectionNodeLabel.vue"
import {
  LayersOutline,
  ServerOutline,
  AddCircleOutline,
  nodeTypeIconMap,
  nodeTypeFallbackIcon,
} from "@/lib/icons"
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
} from "@/bindings/github.com/felixdotgo/querybox/services/app"

// declare events emitted by this component
const emit = defineEmits([
  "connection-selected",
  "query-result",
  "connection-dblclick",
])

const router = useRouter()
async function openConnections() {
  try {
    await ShowConnectionsWindow()
    return
  } catch (err) {
    router.push("/connections")
  }
}

// panel state -------------------------------------------------------------
const connections = ref([])
const filter = ref("")
const connectionTrees = ref({})
const selectedConnection = ref(null)
const expandedKeys = ref([])
const deleteModal = ref({ visible: false, conn: null })

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
    connectionTrees.value = {}
  } catch (err) {
    console.error("ListConnections", err)
    connections.value = []
    connectionTrees.value = {}
  }
}

function handleSelect(keys, options, meta) {
  const key = meta?.node?.key ?? (Array.isArray(keys) ? keys[0] : keys)
  console.debug("handleSelect key", key, "meta.node", meta?.node)
  if (key == null) return

  // top‑level connection selected
  const conn = connections.value.find((c) => c.id === key)
  if (conn) {
    selectedConnection.value = conn
    // do not automatically load the tree; user must hit the 'Connect'
    // button now. this keeps connection selection lightweight and avoids
    // surprise network calls on click.
    emit("connection-selected", conn)
    return
  }

  // determine which connection the clicked tree node belongs to.  normally
  // `selectedConnection` will already be set (user clicked the connection
  // parent first), but if not we attempt to infer it by walking the cached
  // tree data. this makes the behaviour a little more forgiving.
  let parentConn = selectedConnection.value
  const node = meta?.node
  if (!parentConn) {
    for (const c of connections.value) {
      const nodes = connectionTrees.value[c.id] || []
      const finder = (list) => {
        for (const n of list) {
          if (n.key === key) return true
          if (n.children && finder(n.children)) return true
        }
        return false
      }
      if (finder(nodes)) {
        parentConn = c
        // ensure we have a tree available so runTreeAction has metadata
        fetchTreeFor(c)
        break
      }
    }
  }

  // execute the first action for leaf nodes (nodes with no children).
  // non-leaf nodes (databases, schemas, etc.) are only used for navigation
  // so clicking them should not trigger a query.
  const isLeaf = !node?.children || node.children.length === 0
  if (parentConn && node && isLeaf && node.actions && node.actions.length > 0) {
    const act = node.actions[0]
    runTreeAction(parentConn, act, node)
  }

}

function handleConnectionDblclick(conn) {
  if (!conn) return
  selectedConnection.value = conn
  const copy = { ...connectionTrees.value }
  delete copy[conn.id]
  connectionTrees.value = copy
  checkConnection(conn)
  // tree load remains tied to the explicit connect button so we don't
  // auto-fetch here
  emit("connection-dblclick", conn)
}

function getNodeProps(node) {
  const props = {}
  const conn = connections.value.find((c) => c.id === node.key)
  if (conn) {
    // double‑click on a connection header opens the connections window
    props.onDblclick = (e) => {
      e.stopPropagation()
      handleConnectionDblclick(conn)
    }
  }

  return props
}

function renderLabel({ option }) {
  const conn = connections.value.find((c) => c.id === option.key)
  // non-connection nodes (driver group headers, database/table nodes) just show the label
  if (!conn) return option.label

  return h(ConnectionNodeLabel, {
    label: option.label,
    hasTree: !!connectionTrees.value[conn.id],
    onConnect() {
      if (connectionTrees.value[conn.id]) {
        const copy = { ...connectionTrees.value }
        delete copy[conn.id]
        connectionTrees.value = copy
      }
      fetchTreeFor(conn)
    },
    onDelete() {
      deleteModal.value = { visible: true, conn }
    },
    onDblclick() {
      handleConnectionDblclick(conn)
    },
  })
}

function renderPrefix({ option }) {
  let icon
  if (String(option.key).startsWith("driver:")) {
    icon = LayersOutline
  } else if (connections.value.find((c) => c.id === option.key)) {
    icon = ServerOutline
  } else {
    icon = nodeTypeIconMap[option.node_type] ?? nodeTypeFallbackIcon
  }
  return h(NIcon, { size: 14 }, { default: () => h(icon) })
}

async function confirmDelete() {
  const conn = deleteModal.value.conn
  if (!conn) return
  try {
    // Backend emits connection:deleted on success; the event handler cleans up state.
    await DeleteConnection(conn.id)
  } catch (err) {
    console.error("DeleteConnection", err)
  } finally {
    deleteModal.value = { visible: false, conn: null }
  }
}

async function runTreeAction(conn, action, node) {
  // mark invocation time before hitting the network so we can compare
  // request ordering regardless of return speed.
  const invocationVersion = Date.now()

  // Hoist stable identifiers so they are available in the catch block too.
  const tabKey = conn.id + ":" + (node && node.key ? node.key : action.query || invocationVersion)
  let title = (node && node.key) || action.title || action.query || "Query"
  title = title.split(":").pop()

  try {
    const cred = await GetCredential(conn.id)
    const params = {}
    if (cred) params.credential_blob = cred
    let queryToRun = action.query || ""
    if (
      action.type === "select" &&
      /^\s*select\b/i.test(queryToRun) &&
      !/\blimit\b/i.test(queryToRun)
    ) {
      queryToRun = queryToRun.trim() + " LIMIT 100"
    }

    const res = await ExecTreeAction(conn.driver_type, params, queryToRun)

    // regardless of query type we unwrap and normalise any result that came
    // back; the workspace will decide how to render it (or just show an
    // error if `res.error` is set).  this makes behaviour consistent for
    // non-SELECT actions like 'USE' if we ever want to display feedback.
    let payload = res.result || {}
    if (payload && payload.Payload) {
      payload = payload.Payload
    }

    if (payload.Sql) payload = payload.Sql
    else if (payload.Document) payload = payload.Document
    else if (payload.Kv) payload = payload.Kv

    // capitalised keys (protojson output) confuse the viewer; lowercase
    // everything once so callers don't have to care.
    const normalizeKeys = (obj) => {
      if (!obj || typeof obj !== "object") return obj
      const out = {}
      for (const key of Object.keys(obj)) {
        const lower = key.charAt(0).toLowerCase() + key.slice(1)
        out[lower] = obj[key]
      }
      return out
    }
    payload = normalizeKeys(payload)

    // use the version we captured at the start; the response time may
    // not reflect request order, so this guarantees the later-initiated
    // query cannot be accidentally overwritten by an earlier one.
    const version = invocationVersion
    console.debug("runTreeAction result", action, queryToRun, res, payload, tabKey, version)

    if (res.error) {
      emit("query-result", title, null, res.error, tabKey, version)
    } else {
      emit("query-result", title, payload, null, tabKey, version)
    }
  } catch (err) {
    console.error("ExecTreeAction", conn.id, err)
    // Surface the error in the workspace tab so it is visible to the user.
    // The backend already emits an app:log event for these failures;
    // opening an error tab gives additional in-context feedback.
    emit("query-result", title, null, err?.message || String(err), tabKey, invocationVersion)
  }
}

async function checkConnection(conn) {
  try {
    const cred = await GetCredential(conn.id)
    const params = {}
    if (cred) params.credential_blob = cred
    await ExecPlugin(conn.driver_type, params, "SELECT 1")
  } catch (err) {
    console.error("connection check", conn.id, err)
  }
}

async function fetchTreeFor(conn) {
  if (connectionTrees.value[conn.id]) {
    return
  }
  try {
    const cred = await GetCredential(conn.id)
    const params = {}
    if (cred) params.credential_blob = cred
    const resp = await GetConnectionTree(conn.driver_type, params)
    connectionTrees.value = {
      ...connectionTrees.value,
      [conn.id]: resp.nodes || [],
    }
    if (!expandedKeys.value.includes(conn.id)) {
      expandedKeys.value.push(conn.id)
    }
  } catch (err) {
    console.error("GetConnectionTree", conn.id, err)
    connectionTrees.value = { ...connectionTrees.value, [conn.id]: [] }
  }
}

// watch to ensure driver groups expanded when connections reload
watch(connections, () => {
  expandedKeys.value = defaultExpandedKeys.value
})

// initialize
loadConnections()

// Backend domain events — frontend only listens, never emits these topics.

// connection:created → prepend the new connection (avoids a full re-fetch).
const offConnectionCreated = Events.On("connection:created", (event) => {
  const conn = (event?.data ?? event)?.connection
  if (!conn) return
  connections.value = [conn, ...connections.value]
})

// connection:deleted → reactively remove from local state.
const offConnectionDeleted = Events.On("connection:deleted", (event) => {
  const id = (event?.data ?? event)?.id
  if (!id) return
  connections.value = connections.value.filter((c) => c.id !== id)
  delete connectionTrees.value[id]
  if (selectedConnection.value?.id === id) selectedConnection.value = null
  expandedKeys.value = expandedKeys.value.filter((k) => k !== id)
})

onUnmounted(() => {
  if (offConnectionCreated) offConnectionCreated()
  if (offConnectionDeleted) offConnectionDeleted()
})
</script>
