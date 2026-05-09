import type { ComputedRef, Ref } from 'vue'
import type { Connection, TreeAction, TreeNode } from '@/lib/types'
import { useDialog, useNotification } from 'naive-ui'
import { ref } from 'vue'
import {
  DeleteConnection,
  GetCredential,
} from '@/bindings/github.com/felixdotgo/querybox/services/connectionservice'
import {
  ExecPlugin,
  ExecTreeAction,
} from '@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager'
import { extractDatabase } from '@/lib/nodeKey'

/** Node types that immediately trigger a select action on click. */
const INSTANT_SELECT_TYPES = new Set(['table', 'collection', 'key', 'view', 'foreign-table'])

/** Action types that open a user-input form before execution. */
const PROMPT_ACTION_TYPES = new Set(['create-database', 'create-table'])

/** Action types that require a destructive confirmation dialog. */
const DESTRUCTIVE_ACTION_TYPES = new Set(['drop-database', 'drop-table', 'drop-collection'])

function actionTypeOf(action: TreeAction | null | undefined): string {
  return String(action?.type ?? action?.kind ?? action?.id ?? '')
}

function nodeKindOf(node: TreeNode | null | undefined): string {
  return String(node?.kind ?? node?.node_type ?? '')
}

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

  const deleteModal = ref<{ visible: boolean, conn: Connection | null }>({ visible: false, conn: null })
  const actionModal = ref<{ visible: boolean, action: TreeAction | null, conn: Connection | null, node: TreeNode | null }>({
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
      const res = await ExecPlugin(conn.driver_type, params, 'SELECT 1', {})
      if (res && (res as { error?: string }).error) {
        throw new Error((res as { error?: string }).error)
      }
    }
    catch (err: unknown) {
      console.error('connection check', conn.id, err)
      notification.error({
        title: 'Connection failed',
        content: (err as Error)?.message || String(err),
        duration: 5000,
      })
      throw err
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
        if (!res)
          return
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
      const actionType = actionTypeOf(action)
      if (
        actionType === 'select'
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
      if (!res)
        return

      let payload: unknown = res.result || {}
      if (payload && typeof payload === 'object' && 'Payload' in payload) {
        payload = (payload as { Payload?: unknown }).Payload ?? payload
      }

      if (payload && typeof payload === 'object' && 'Sql' in payload)
        payload = (payload as { Sql?: unknown }).Sql ?? payload
      else if (payload && typeof payload === 'object' && 'Document' in payload)
        payload = (payload as { Document?: unknown }).Document ?? payload
      else if (payload && typeof payload === 'object' && 'Kv' in payload)
        payload = (payload as { Kv?: unknown }).Kv ?? payload

      const normalizeKeys = (obj: unknown): unknown => {
        if (!obj || typeof obj !== 'object')
          return obj
        const out: Record<string, unknown> = {}
        for (const key of Object.keys(obj)) {
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
    const actionType = actionTypeOf(action)
    if (PROMPT_ACTION_TYPES.has(actionType)) {
      actionModal.value = { visible: true, action, conn, node }
      return
    }

    if (DESTRUCTIVE_ACTION_TYPES.has(actionType)) {
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
    meta: { node?: TreeNode & { key: string, _connectionId?: string } } | undefined,
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

    const nodeType = nodeKindOf(node)

    if (nodeType === 'action' && node.actions && node.actions.length > 0) {
      handleAction(parentConn, node.actions[0], node)
      return
    }

    if (INSTANT_SELECT_TYPES.has(String(nodeType))) {
      const selectAction = node.actions?.find(a => actionTypeOf(a) === 'select')
      if (selectAction)
        handleAction(parentConn, selectAction, node)
      return
    }

    const hasChildren = Array.isArray(node.children) && node.children.length > 0
    const hasSelectAction = node.actions?.some(a => actionTypeOf(a) === 'select')
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
    checkConnection(conn).catch(() => { /* notification already shown */ })
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
