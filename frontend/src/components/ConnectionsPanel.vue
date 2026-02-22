<template>
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

    <!-- delete confirmation overlay -->
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
  </div>
</template>

<script setup>
import { ref, computed, h, watch, onMounted, onUnmounted } from "vue"
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
} from "@/bindings/github.com/felixdotgo/querybox/services/app"

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
  if (key == null) return

  const conn = connections.value.find((c) => c.id === key)
  if (conn) {
    selectedConnection.value = conn
    fetchTreeFor(conn)
    emit("connection-selected", conn)
    return
  }

  const parentConn = selectedConnection.value
  const node = meta?.node
  if (parentConn && node && node.actions && node.actions.length > 0) {
    const act = node.actions[0]
    runTreeAction(parentConn, act)
  }
}

function handleConnectionDblclick(conn) {
  if (!conn) return
  selectedConnection.value = conn
  const copy = { ...connectionTrees.value }
  delete copy[conn.id]
  connectionTrees.value = copy
  checkConnection(conn)
  fetchTreeFor(conn)
  emit("connection-dblclick", conn)
}

function getNodeProps(node) {
  const conn = connections.value.find((c) => c.id === node.key)
  if (!conn) return {}
  return {
    onDblclick(e) {
      e.stopPropagation()
      handleConnectionDblclick(conn)
    },
  }
}

function renderLabel({ option }) {
  const conn = connections.value.find((c) => c.id === option.key)
  if (!conn) {
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
    connections.value = connections.value.filter((c) => c.id !== conn.id)
    delete connectionTrees.value[conn.id]
    if (selectedConnection.value?.id === conn.id)
      selectedConnection.value = null
    expandedKeys.value = expandedKeys.value.filter((k) => k !== conn.id)
  } catch (err) {
    console.error("DeleteConnection", err)
  } finally {
    deleteModal.value = { visible: false, conn: null }
  }
}

async function runTreeAction(conn, action) {
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
    if (
      action.type === "select" &&
      /^\s*select\b/i.test(action.query || "")
    ) {
      const title = action.title || action.query || "Query"
      emit("query-result", title, res.result, res.error)
    }
  } catch (err) {
    console.error("ExecTreeAction", conn.id, err)
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

// reload on connection saved
const offConnectionSaved = Events.On("connection:saved", () => {
  loadConnections()
})

onUnmounted(() => {
  if (offConnectionSaved) offConnectionSaved()
})
</script>
