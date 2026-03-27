import { ref } from 'vue'
import type { ComputedRef, Ref } from 'vue'
import { useDialog, useNotification } from 'naive-ui'
import {
  DeleteConnection,
  GetCredential,
} from '@/bindings/github.com/felixdotgo/querybox/services/connectionservice'
import {
  ExecPlugin,
  ExecTreeAction,
} from '@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager'
import { extractDatabase } from '@/lib/nodeKey'
import type { Connection, TreeAction, TreeNode } from '@/lib/types'

/** Node types that immediately trigger a select action on click. */
const INSTANT_SELECT_TYPES = new Set(['table', 'collection', 'key', 'view', 'foreign-table'])

/** Action types that open a user-input form before execution. */
const PROMPT_ACTION_TYPES = new Set(['create-database', 'create-table'])

/** Action types that require a destructive confirmation dialog. */
const DESTRUCTIVE_ACTION_TYPES = new Set(['drop-database', 'drop-table', 'drop-collection'])

interface UseTreeActionsOptions {
  connections: Ref<Connection[]>
  connectionTrees: Record<string, TreeNode[]>
  schemaCache: Record<string, unknown>
  expandedKeys: Ref<string[]>
  loadingNodes: Ref<Record<string, boolean>>
  connecting: Ref<Record<string, boolean>>
  selectedConnection: Ref<Connection | null>
  pluginCaps: ComputedRef<Record<string, string[]>>
  loadConnectionTree: (conn: Connection) => Promise<void>
  emit: (event: string, ...args: unknown[]) => void
}

export function useTreeActions({
  connections,
  connectionTrees,
  schemaCache,
  expandedKeys,
  loadingNodes,
  connecting,
  selectedConnection,
  pluginCaps,
  loadConnectionTree,
  emit,
}: UseTreeActionsOptions) {
  const dialog = useDialog()
  const notification = useNotification()

  const deleteModal = ref<{ visible: boolean; conn: Connection | null }>({ visible: false, conn: null })
  const actionModal = ref<{ visible: boolean; action: TreeAction | null; conn: Connection | null; node: TreeNode | null }>({
    visible: false,
    action: null,
    conn: null,
    node: null,
  })

  async function fetchTreeFor(conn: Connection) {
    if (!conn)
      return
    connecting.value[conn.id] = true
    loadingNodes.value[conn.id] = true
    try {
      await loadConnectionTree(conn)
      if (!expandedKeys.value.includes(conn.id)) {
        expandedKeys.value = [...expandedKeys.value, conn.id]
      }
    }
    catch (err: unknown) {
      console.error('fetchTreeFor', conn.id, err)
      notification.error({ title: 'Connection failed', content: (err as Error)?.message || String(err), duration: 5000 })
    }
    finally {
      delete connecting.value[conn.id]
      delete loadingNodes.value[conn.id]
    }
  }

  async function checkConnection(conn: Connection) {
    try {
      const cred = await GetCredential(conn.id)
      const params: Record<string, string> = {}
      if (cred)
        params.credential_blob = cred
      await ExecPlugin(conn.driver_type, params, 'SELECT 1', {})
    }
    catch (err: unknown) {
      console.error('connection check', conn.id, err)
    }
  }

  async function runTreeAction(conn: Connection, action: TreeAction, node: TreeNode | null, extras: Record<string, unknown> = {}) {
    const nodeKeyForSpinner = node?.key ?? null
    if (nodeKeyForSpinner) {
      loadingNodes.value[nodeKeyForSpinner] = true
    }

    const invocationVersion = Date.now()

    const nodeKey = node?.key ?? (action.query || String(invocationVersion))
    const tabKey = (typeof nodeKey === 'string' && nodeKey.startsWith(`${conn.id}:`))
      ? nodeKey
      : `${conn.id}:${nodeKey}`
    let title = (node?.key) || action.title || action.query || 'Query'
    title = title.split(':').pop() ?? title

    if (!action.new_tab) {
      try {
        const cred = await GetCredential(conn.id)
        const params: Record<string, string> = {}
        if (cred)
          params.credential_blob = cred
        if (node?.key && typeof node.key === 'string') {
          const db = extractDatabase(conn.id, node.key)
          if (db)
            params.database = db
        }
        const res = await ExecTreeAction(
          conn.driver_type,
          params,
          action.query || '',
          (extras.options as Record<string, string>) || ((extras.explain) ? { 'explain-query': 'yes' } : {}),
        )
        if (!res) return
        if (res.error) {
          console.error('runTreeAction [hidden]', action.type, res.error)
          notification.error({ title: 'Action failed', content: res.error, duration: 5000 })
        }
        else {
          delete connectionTrees[conn.id]
          delete schemaCache[conn.id]
          fetchTreeFor(conn)
        }
      }
      catch (err: unknown) {
        console.error('runTreeAction [hidden] error', action.type, (err as Error)?.message || err)
        notification.error({ title: 'Action failed', content: (err as Error)?.message || String(err), duration: 5000 })
      }
      return
    }

    try {
      const cred = await GetCredential(conn.id)
      const params: Record<string, string> = {}
      if (cred)
        params.credential_blob = cred
      if (node?.key && typeof node.key === 'string') {
        const db = extractDatabase(conn.id, node.key)
        if (db)
          params.database = db
      }
      let queryToRun = action.query || ''
      if (
        action.type === 'select'
        && /^\s*select\b/i.test(queryToRun)
        && !/\blimit\b/i.test(queryToRun)
      ) {
        queryToRun = `${queryToRun.trim()} LIMIT 100`
      }

      const res = await ExecTreeAction(
        conn.driver_type,
        params,
        queryToRun,
        (extras.options as Record<string, string>) || ((extras.explain) ? { 'explain-query': 'yes' } : {}),
      )
      if (!res) return

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      let payload: any = res.result || {}
      if (payload?.Payload) {
        payload = payload.Payload
      }

      if (payload.Sql)
        payload = payload.Sql
      else if (payload.Document)
        payload = payload.Document
      else if (payload.Kv)
        payload = payload.Kv

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const normalizeKeys = (obj: any): any => {
        if (!obj || typeof obj !== 'object')
          return obj
        const out: Record<string, unknown> = {}
        for (const key of Object.keys(obj as object)) {
          const lower = key.charAt(0).toLowerCase() + key.slice(1)
          out[lower] = (obj as Record<string, unknown>)[key]
        }
        return out
      }
      payload = normalizeKeys(payload)

      const context = {
        conn,
        action,
        node,
        capabilities: pluginCaps.value[conn.driver_type] || [],
        ...extras,
      }

      if (res.error) {
        emit('query-result', { title, result: null, error: res.error, tabKey, version: invocationVersion, context })
      }
      else {
        emit('query-result', { title, result: payload, error: null, tabKey, version: invocationVersion, context })
      }
    }
    catch (err: unknown) {
      console.error('ExecTreeAction', conn.id, err)
      const context = { conn, action, node }
      emit('query-result', { title, result: null, error: (err as Error)?.message || String(err), tabKey, version: invocationVersion, context })
    }
    finally {
      if (nodeKeyForSpinner) {
        delete loadingNodes.value[nodeKeyForSpinner]
      }
    }
  }

  function handleAction(conn: Connection, action: TreeAction, node: TreeNode | null) {
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

  function onActionModalSubmit(modifiedQuery: string) {
    const { conn, action, node } = actionModal.value
    if (!conn || !action)
      return
    runTreeAction(conn, { ...action, query: modifiedQuery }, node)
  }

  function handleSelect(
    keys: string[],
    _options: unknown,
    meta: { node?: TreeNode & { key: string; _connectionId?: string } } | undefined,
  ) {
    const key = meta?.node?.key ?? (Array.isArray(keys) ? keys[0] : keys)
    if (key == null)
      return

    const conn = connections.value.find(c => c.id === key)
    if (conn) {
      selectedConnection.value = conn
      if (!connectionTrees[conn.id]) {
        delete connectionTrees[conn.id]
        delete schemaCache[conn.id]
        fetchTreeFor(conn)
        emit('connection-selected', conn)
        emit('connection-opened', conn)
      }
      else {
        const idx = expandedKeys.value.indexOf(conn.id)
        if (idx === -1) {
          expandedKeys.value = [...expandedKeys.value, conn.id]
        }
        else {
          expandedKeys.value = expandedKeys.value.filter(k => k !== conn.id)
        }
      }
      return
    }

    const node = meta?.node
    if (!node)
      return

    const parentConn = node._connectionId
      ? connections.value.find(c => c.id === node._connectionId)
      : selectedConnection.value ?? undefined
    if (!parentConn)
      return

    const nodeType = node.node_type

    if (nodeType === 'action' && node.actions && node.actions.length > 0) {
      handleAction(parentConn, node.actions[0], node)
      return
    }

    if (INSTANT_SELECT_TYPES.has(String(nodeType))) {
      const selectAction = node.actions?.find(a => a.type === 'select')
      if (selectAction)
        handleAction(parentConn, selectAction, node)
      return
    }

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

  function handleConnectionDblclick(conn: Connection) {
    if (!conn)
      return
    selectedConnection.value = conn
    delete connectionTrees[conn.id]
    delete schemaCache[conn.id]
    checkConnection(conn)
    emit('connection-opened', conn)
  }

  async function confirmDelete() {
    const conn = deleteModal.value.conn
    if (!conn)
      return
    try {
      await DeleteConnection(conn.id)
    }
    catch (err: unknown) {
      console.error('DeleteConnection', err)
    }
    finally {
      deleteModal.value = { visible: false, conn: null }
    }
  }

  return {
    deleteModal,
    actionModal,
    runTreeAction,
    fetchTreeFor,
    checkConnection,
    handleAction,
    handleSelect,
    handleConnectionDblclick,
    onActionModalSubmit,
    confirmDelete,
  }
}
