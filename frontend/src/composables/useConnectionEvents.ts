import type { Ref } from 'vue'
import type { Connection } from '@/lib/types'
import { Events } from '@wailsio/runtime'
import { onUnmounted } from 'vue'
import { ListConnections } from '@/bindings/github.com/felixdotgo/querybox/services/connectionservice'

interface ConnectionEventsOptions {
  connections: Ref<Connection[]>
  connectionTrees: Record<string, unknown>
  schemaCache: Record<string, unknown>
  selectedConnection: Ref<Connection | null>
  expandedKeys: Ref<string[]>
  loadingNodes: Ref<Record<string, boolean>>
  filter: Ref<string>
}

/**
 * Subscribes to backend connection domain events (created, updated, deleted)
 * and keeps local state in sync without disturbing already-open connections.
 *
 * - loadConnections() refreshes the array only (used for initial load).
 * - created/updated/deleted handlers mutate the array and per-id caches in place,
 *   leaving the active connection's tree/schema cache untouched.
 */
export function useConnectionEvents(opts: ConnectionEventsOptions) {
  const {
    connections,
    connectionTrees,
    schemaCache,
    selectedConnection,
    expandedKeys,
    loadingNodes,
    filter,
  } = opts

  async function loadConnections() {
    try {
      connections.value = (await ListConnections()) || []
    }
    catch (err) {
      console.error('ListConnections', err)
      connections.value = []
    }
  }

  const offConnectionCreated = Events.On('connection:created', (event: any) => {
    const conn = (event?.data ?? event)?.connection as Connection | undefined
    if (!conn?.id)
      return
    if (!connections.value.some(c => c.id === conn.id))
      connections.value = [...connections.value, conn]
    filter.value = ''
  })

  const offConnectionDeleted = Events.On('connection:deleted', (event: any) => {
    const id = (event?.data ?? event)?.id
    if (!id)
      return
    connections.value = connections.value.filter(c => c.id !== id)
    delete connectionTrees[id]
    delete schemaCache[id]
    if (selectedConnection.value?.id === id)
      selectedConnection.value = null
    expandedKeys.value = expandedKeys.value.filter(k => k !== id)
    Object.keys(loadingNodes.value).forEach((k) => {
      if (k.startsWith(`${id}:`))
        delete loadingNodes.value[k]
    })
  })

  const offConnectionUpdated = Events.On('connection:updated', (event: any) => {
    const conn = (event?.data ?? event)?.connection as Connection | undefined
    const id = conn?.id
    if (!id)
      return
    const idx = connections.value.findIndex(c => c.id === id)
    if (idx !== -1) {
      const next = connections.value.slice()
      next[idx] = conn
      connections.value = next
    }
    delete connectionTrees[id]
    delete schemaCache[id]
  })

  onUnmounted(() => {
    if (offConnectionCreated)
      offConnectionCreated()
    if (offConnectionDeleted)
      offConnectionDeleted()
    if (offConnectionUpdated)
      offConnectionUpdated()
  })

  return { loadConnections }
}
