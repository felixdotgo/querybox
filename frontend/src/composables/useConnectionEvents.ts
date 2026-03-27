import { Events } from '@wailsio/runtime'
import { onUnmounted, type Ref } from 'vue'
import { ListConnections } from '@/bindings/github.com/felixdotgo/querybox/services/connectionservice'
import type { Connection } from '@/lib/types'

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
 * and keeps local state in sync. Automatically unsubscribes on unmount.
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
      Object.keys(connectionTrees).forEach(k => delete connectionTrees[k])
      Object.keys(schemaCache).forEach(k => delete schemaCache[k])
    }
    catch (err) {
      console.error('ListConnections', err)
      connections.value = []
      Object.keys(connectionTrees).forEach(k => delete connectionTrees[k])
      Object.keys(schemaCache).forEach(k => delete schemaCache[k])
    }
  }

  const offConnectionCreated = Events.On('connection:created', async (event: any) => {
    const conn = (event?.data ?? event)?.connection
    if (!conn)
      return
    try {
      await loadConnections()
      filter.value = ''
    }
    catch (err) {
      console.error('connection:created handler loadConnections', err)
    }
  })

  const offConnectionDeleted = Events.On('connection:deleted', async (event: any) => {
    const id = (event?.data ?? event)?.id
    if (!id)
      return
    try {
      await loadConnections()
    }
    catch (err) {
      console.error('connection:deleted handler loadConnections', err)
    }
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

  const offConnectionUpdated = Events.On('connection:updated', async (event: any) => {
    const id = (event?.data ?? event)?.connection?.id
    if (id) {
      delete connectionTrees[id]
      delete schemaCache[id]
    }
    try {
      await loadConnections()
    }
    catch (err) {
      console.error('connection:updated handler loadConnections', err)
    }
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
