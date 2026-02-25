<script setup>
import { Events } from '@wailsio/runtime'
import { NButton, NIcon, useDialog } from 'naive-ui'
import { computed, h, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import {
  ShowConnectionsWindow,
} from '@/bindings/github.com/felixdotgo/querybox/services/app'
import {
  DeleteConnection,
  GetCredential,
  ListConnections,
} from '@/bindings/github.com/felixdotgo/querybox/services/connectionservice'
// @ts-expect-error: may be generated after adding new methods
import {
  ExecPlugin,
  ExecTreeAction,
} from '@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager'
import ActionFormModal from '@/components/ActionFormModal.vue'
import ConnectionNodeLabel from '@/components/ConnectionNodeLabel.vue'
import ConnectionTreeNodeLabel from '@/components/ConnectionTreeNodeLabel.vue'
import { useConnectionTree } from '@/composables/useConnectionTree'
import {
  AddCircleOutline,
  nodeTypeFallbackIcon,
  nodeTypeIconMap,
  ServerOutline,
} from '@/lib/icons'

const props = defineProps({
  activeConnectionId: { type: String, default: null },
})

// declare events emitted by this component
const emit = defineEmits([
  'connection-selected',
  'query-result',
  'connection-opened',
])

const dialog = useDialog()

const router = useRouter()
async function openConnections() {
  try {
    await ShowConnectionsWindow()
  }
  catch {
    router.push('/connections')
  }
}

// panel state -------------------------------------------------------------
const treeScrollRef = ref(null)
const isScrolled = ref(false)
const connections = ref([])
const filter = ref('')
// connectionTrees replaced by shared cache from composable
const { cache: connectionTrees, load: loadConnectionTree } = useConnectionTree()
const selectedConnection = ref(null)
const expandedKeys = ref([])
const deleteModal = ref({ visible: false, conn: null })
const actionModal = ref({ visible: false, action: null, conn: null, node: null })

const defaultExpandedKeys = computed(() => {
  // include every connection id so that roots are always visible when
  // expansion state is reset.  `connectionTrees` only contains entries for
  // connections whose tree has been loaded, so combining both ensures the
  // first connection never vanishes after a search.
  const ids = new Set(Object.keys(connectionTrees))
  connections.value.forEach(c => ids.add(c.id))
  return Array.from(ids)
})

/**
 * Maps proto NodeType enum integers (as serialized by encoding/json) to the
 * lowercase strings expected by INSTANT_SELECT_TYPES, nodeTypeIconMap, and
 * the node-type guards throughout this component.
 */
const NODE_TYPE_ENUM_MAP = {
  1: 'database',
  2: 'table',
  3: 'column',
  4: 'schema',
  5: 'view',
  6: 'action',
  7: 'collection',
  8: 'key', // plugin-local extension for key-value store leaf nodes (e.g. Redis)
}

/**
 * Recursively stamps every node (and its descendants) with the owning
 * connection id.  This lets the three parentConn-finder loops below resolve
 * the correct connection in O(1) instead of scanning every loaded tree,
 * which previously picked the wrong driver when two connections shared a
 * node key such as "__server__".
 *
 * Also normalises node_type from proto enum integers (e.g. 2) to the
 * lowercase strings the rest of the component expects (e.g. "table").
 */
function tagWithConnId(nodes, connId) {
  return nodes.map((n) => {
    const nodeType = typeof n.node_type === 'number'
      ? (NODE_TYPE_ENUM_MAP[n.node_type] ?? null)
      : n.node_type
    return {
      ...n,
      key: `${connId}:${n.key}`,
      _connectionId: connId,
      node_type: nodeType,
      children: n.children ? tagWithConnId(n.children, connId) : n.children,
    }
  })
}

const treeData = computed(() => {
  return (connections.value || []).map((cc) => {
    const extra = tagWithConnId(connectionTrees[cc.id] || [], cc.id)
    return { key: cc.id, label: cc.name, children: extra.length ? extra : undefined }
  })
})

// Retain original node references — only slice the top-level connections array.
// Child nodes are never cloned so all tree interactions remain intact.
const filteredTreeData = computed(() => {
  const q = (filter.value || '').toLowerCase().trim()
  if (!q)
    return treeData.value
  return treeData.value.filter(node =>
    (node.label || '').toLowerCase().includes(q),
  )
})

async function loadConnections() {
  try {
    connections.value = (await ListConnections()) || []
    // clear cache when whole list is reloaded; avoids stale entries
    Object.keys(connectionTrees).forEach(k => delete connectionTrees[k])
  }
  catch (err) {
    console.error('ListConnections', err)
    connections.value = []
    Object.keys(connectionTrees).forEach(k => delete connectionTrees[k])
  }
}

// Node types that are data-bearing leaves: clicking them should immediately
// open a tab showing the data via their "select" action.
const INSTANT_SELECT_TYPES = new Set(['table', 'collection', 'key', 'view', 'foreign-table'])

function handleSelect(keys, options, meta) {
  const key = meta?.node?.key ?? (Array.isArray(keys) ? keys[0] : keys)
  if (key == null)
    return

  // top‑level connection selected
  const conn = connections.value.find(c => c.id === key)
  if (conn) {
    selectedConnection.value = conn
    // do not automatically load the tree; user must hit the 'Connect'
    // button now. this keeps connection selection lightweight and avoids
    // surprise network calls on click.
    emit('connection-selected', conn)
    return
  }

  // Tree node clicked — resolve the owning connection.
  const node = meta?.node
  if (!node)
    return

  const parentConn = node._connectionId
    ? connections.value.find(c => c.id === node._connectionId)
    : selectedConnection.value
  if (!parentConn)
    return

  const nodeType = node.node_type

  // "action" leaf nodes (e.g. "New database", "New table"): fire the
  // first action immediately on click without needing to hover.
  if (nodeType === 'action' && node.actions?.length > 0) {
    handleAction(parentConn, node.actions[0], node)
    return
  }

  // Data-bearing leaf nodes: fire the "select" action immediately to
  // open a result tab, giving instant feedback on single click.
  if (INSTANT_SELECT_TYPES.has(nodeType)) {
    const selectAction = node.actions?.find(a => a.type === 'select')
    if (selectAction)
      handleAction(parentConn, selectAction, node)
    return
  }

  // Container nodes (database, schema, collection, …) that have children
  // but no select action should expand/collapse on click instead of doing
  // nothing.  This gives a natural feel when browsing the tree.
  const hasChildren = Array.isArray(node.children) && node.children.length > 0
  const hasSelectAction = node.actions?.some(a => a.type === 'select')
  if (hasChildren && !hasSelectAction) {
    const idx = expandedKeys.value.indexOf(node.key)
    if (idx === -1) {
      expandedKeys.value = [...expandedKeys.value, node.key]
    }
    else {
      expandedKeys.value = expandedKeys.value.filter(k => k !== node.key)
    }
  }
}

function handleConnectionDblclick(conn) {
  if (!conn)
    return
  selectedConnection.value = conn
  // remove cached tree so that a fresh fetch happens on reconnect
  delete connectionTrees[conn.id]
  checkConnection(conn)
  // tree load remains tied to the explicit connect button so we don't
  // auto-fetch here
  emit('connection-opened', conn)
}

function getNodeProps(node) {
  const props = {}
  const conn = connections.value.find(c => c.id === node.key)
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
  const conn = connections.value.find(c => c.id === option.key)

  // "action" leaf nodes (New database, New table, …) are rendered as secondary
  // buttons with their icon embedded; clicking the row fires the action via handleSelect.
  if (!conn && option.node_type === 'action') {
    const actionIcon = nodeTypeIconMap[option.node_type] ?? nodeTypeFallbackIcon
    return h(
      NButton,
      { size: 'tiny', secondary: true, style: 'margin: 1px 0', type: 'primary' },
      {
        icon: () => h(NIcon, { size: 14 }, { default: () => h(actionIcon) }),
        default: () => option.label,
      },
    )
  }

  // non-connection nodes with plugin-defined actions: render action buttons on hover
  if (!conn && option.actions && option.actions.length > 0) {
    return h(ConnectionTreeNodeLabel, {
      label: option.label,
      actions: option.actions,
      onAction(action) {
        const parentConn = option._connectionId
          ? connections.value.find(c => c.id === option._connectionId)
          : selectedConnection.value
        if (parentConn)
          handleAction(parentConn, action, option)
      },
    })
  }

  // driver group headers and connection-less plain nodes just show the label
  if (!conn)
    return option.label

  return h(ConnectionNodeLabel, {
    label: option.label,
    hasTree: !!connectionTrees[conn.id],
    isActive: props.activeConnectionId === conn.id,
    onConnect() {
      if (connectionTrees[conn.id]) {
        delete connectionTrees[conn.id]
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
  // action nodes render their icon inside the button label; skip the prefix.
  if (option.node_type === 'action')
    return null

  let icon
  const conn = connections.value.find(c => c.id === option.key)
  if (conn) {
    icon = ServerOutline
  }
  else {
    icon = nodeTypeIconMap[option.node_type] ?? nodeTypeFallbackIcon
  }

  const iconNode = h(NIcon, { size: 14 }, { default: () => h(icon) })

  if (conn && props.activeConnectionId === conn.id) {
    return h('div', { style: { position: 'relative', display: 'inline-flex' } }, [
      iconNode,
      h('span', {
        style: {
          position: 'absolute',
          bottom: '-2px',
          right: '-3px',
          width: '8px',
          height: '8px',
          borderRadius: '50%',
          backgroundColor: '#22c55e',
          border: '1px solid white',
        },
      }),
    ])
  }

  return iconNode
}

async function confirmDelete() {
  const conn = deleteModal.value.conn
  if (!conn)
    return
  try {
    // Backend emits connection:deleted on success; the event handler cleans up state.
    await DeleteConnection(conn.id)
  }
  catch (err) {
    console.error('DeleteConnection', err)
  }
  finally {
    deleteModal.value = { visible: false, conn: null }
  }
}

/** Action types that require the user to fill in a form before execution. */
const PROMPT_ACTION_TYPES = new Set(['create-database', 'create-table'])

/** Action types that require an explicit destructive confirmation dialog. */
const DESTRUCTIVE_ACTION_TYPES = new Set(['drop-database', 'drop-table', 'drop-collection'])

/**
 * Central dispatcher for node actions.
 * Routes create actions to the input form modal, destructive actions to a
 * confirmation dialog, and everything else straight to runTreeAction.
 */
function handleAction(conn, action, node) {
  if (PROMPT_ACTION_TYPES.has(action.type)) {
    actionModal.value = { visible: true, action, conn, node }
    return
  }

  if (DESTRUCTIVE_ACTION_TYPES.has(action.type)) {
    dialog.error({
      title: action.title ?? 'Confirm action',
      content: `The following query will be executed — this cannot be undone:\n\n${action.query}`,
      positiveText: 'Execute',
      negativeText: 'Cancel',
      onPositiveClick() {
        runTreeAction(conn, action, node)
      },
    })
    return
  }

  runTreeAction(conn, action, node)
}

function onActionModalSubmit(modifiedQuery) {
  const { conn, action, node } = actionModal.value
  if (!conn || !action)
    return
  runTreeAction(conn, { ...action, query: modifiedQuery }, node)
}

async function runTreeAction(conn, action, node) {
  // mark invocation time before hitting the network so we can compare
  // request ordering regardless of return speed.
  const invocationVersion = Date.now()

  // Hoist stable identifiers so they are available in the catch block too.
  const nodeKey = node && node.key ? node.key : (action.query || String(invocationVersion))
  const tabKey = (typeof nodeKey === 'string' && nodeKey.startsWith(`${conn.id}:`))
    ? nodeKey
    : `${conn.id}:${nodeKey}`
  let title = (node && node.key) || action.title || action.query || 'Query'
  title = title.split(':').pop()

  // Actions with new_tab=false run silently in the background: no tab is opened.
  // The result is only logged to the console / backend log stream.
  // After a successful silent action the connection tree is refreshed so
  // any structural change (create-table, etc.) is reflected immediately.
  if (!action.new_tab) {
    try {
      const cred = await GetCredential(conn.id)
      const params = {}
      if (cred)
        params.credential_blob = cred
      const res = await ExecTreeAction(conn.driver_type, params, action.query || '')
      if (res.error) {
        console.error('runTreeAction [hidden]', action.type, res.error)
      }
      else {
        console.debug('runTreeAction [hidden] ok', action.type)
        // Refresh the tree so newly created tables/databases appear.
        delete connectionTrees[conn.id]
        fetchTreeFor(conn)
      }
    }
    catch (err) {
      console.error('runTreeAction [hidden] error', action.type, err?.message || err)
    }
    return
  }

  try {
    const cred = await GetCredential(conn.id)
    const params = {}
    if (cred)
      params.credential_blob = cred
    let queryToRun = action.query || ''
    if (
      action.type === 'select'
      && /^\s*select\b/i.test(queryToRun)
      && !/\blimit\b/i.test(queryToRun)
    ) {
      queryToRun = `${queryToRun.trim()} LIMIT 100`
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

    if (payload.Sql)
      payload = payload.Sql
    else if (payload.Document)
      payload = payload.Document
    else if (payload.Kv)
      payload = payload.Kv

    // capitalised keys (protojson output) confuse the viewer; lowercase
    // everything once so callers don't have to care.
    const normalizeKeys = (obj) => {
      if (!obj || typeof obj !== 'object')
        return obj
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
    console.debug('runTreeAction result', action, queryToRun, res, payload, tabKey, version)

    // Store context to support Refresh functionality.
    const context = { conn, action, node }

    if (res.error) {
      emit('query-result', title, null, res.error, tabKey, version, context)
    }
    else {
      emit('query-result', title, payload, null, tabKey, version, context)
    }
  }
  catch (err) {
    console.error('ExecTreeAction', conn.id, err)
    // Surface the error in the workspace tab so it is visible to the user.
    // The backend already emits an app:log event for these failures;
    // opening an error tab gives additional in-context feedback.
    const context = { conn, action, node }
    emit('query-result', title, null, err?.message || String(err), tabKey, invocationVersion, context)
  }
}

async function checkConnection(conn) {
  try {
    const cred = await GetCredential(conn.id)
    const params = {}
    if (cred)
      params.credential_blob = cred
    await ExecPlugin(conn.driver_type, params, 'SELECT 1')
  }
  catch (err) {
    console.error('connection check', conn.id, err)
  }
}

async function fetchTreeFor(conn) {
  if (!conn)
    return
  await loadConnectionTree(conn)
  if (!expandedKeys.value.includes(conn.id)) {
    expandedKeys.value = [...expandedKeys.value, conn.id]
  }
}

// When filter is cleared, reset scroll so the first connection is visible.
// Naive UI's built-in pattern/filter props handle expanding matching nodes.
watch(filter, (q) => {
  if (!(q || '').trim() && treeScrollRef.value) {
    treeScrollRef.value.scrollTop = 0
  }
})

// initialize
loadConnections()

// Backend domain events — frontend only listens, never emits these topics.

// connection:created → prepend the new connection (avoids a full re-fetch).
const offConnectionCreated = Events.On('connection:created', (event) => {
  const conn = (event?.data ?? event)?.connection
  if (!conn)
    return
  connections.value = [conn, ...connections.value]
})

// connection:deleted → reactively remove from local state.
const offConnectionDeleted = Events.On('connection:deleted', (event) => {
  const id = (event?.data ?? event)?.id
  if (!id)
    return
  connections.value = connections.value.filter(c => c.id !== id)
  delete connectionTrees[id]
  if (selectedConnection.value?.id === id)
    selectedConnection.value = null
  expandedKeys.value = expandedKeys.value.filter(k => k !== id)
})

onUnmounted(() => {
  if (offConnectionCreated)
    offConnectionCreated()
  if (offConnectionDeleted)
    offConnectionDeleted()
})

// Expose runTreeAction to support Refresh from the Workspace panel.
defineExpose({
  runTreeAction,
})
</script>

<template>
  <div class="p-3 h-full flex flex-col gap-3">
    <!-- small toolbar -->
    <div class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <span class="text-lg font-semibold m-0">Connections</span>
      </div>
      <div class="flex">
        <NButton
          size="small"
          type="primary"
          title="New connection"
          @click="openConnections"
        >
          <template #icon>
            <NIcon><AddCircleOutline /></NIcon>
          </template>
        </NButton>
      </div>
    </div>

    <n-input
      v-model:value="filter"
      size="small"
      placeholder="Filter"
    />

    <div
      ref="treeScrollRef"
      class="flex-1 overflow-auto mt-2 px-1 min-h-0 transition-shadow duration-150"
      :class="{ 'shadow-[inset_0_6px_6px_-6px_rgba(0,0,0,0.12)]': isScrolled }"
      @scroll.passive="isScrolled = $event.target.scrollTop > 0"
    >
      <n-tree
        v-model:expanded-keys="expandedKeys"
        show-line
        :data="filteredTreeData"
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

    <!-- action input form (create-database, create-table, …) -->
    <ActionFormModal
      v-model:visible="actionModal.visible"
      :action="actionModal.action"
      @submit="onActionModalSubmit"
    />

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
