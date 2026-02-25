import type { Ref } from 'vue'
import { reactive, ref, watch } from 'vue'
// @ts-expect-error: generated bindings may not yet have typings
import { GetCredential } from '@/bindings/github.com/felixdotgo/querybox/services/connectionservice'
// @ts-expect-error: generated bindings may not yet have typings
import { GetConnectionTree } from '@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager'

// global reactive cache mapping connection id -> nodes array
const treeCache: Record<string, any[]> = reactive({})

// map proto enum numbers to lowercase names; mirrors ConnectionsPanel.NODE_TYPE_ENUM_MAP
const NODE_TYPE_ENUM_MAP: Record<number, string> = {
  1: 'database',
  2: 'table',
  3: 'column',
  4: 'schema',
  5: 'view',
  6: 'action',
  7: 'collection',
  8: 'key',
}

function normalizeNodes(nodes: any[]): any[] {
  return nodes.map((n) => {
    const type = typeof n.node_type === 'number'
      ? NODE_TYPE_ENUM_MAP[n.node_type] || String(n.node_type)
      : n.node_type
    return {
      ...n,
      node_type: type,
      children: Array.isArray(n.children) ? normalizeNodes(n.children) : n.children,
    }
  })
}

/**
 * Composable providing access to cached connection trees and helpers
 * for client-side completion/lookups. When a `conn` object is supplied via
 * the optional ref, the tree will be loaded automatically.  Otherwise the
 * caller may manually invoke `load(conn)`.
 */
export function useConnectionTree(connRef?: Ref<any | null>) {
  const nodes = ref<any[]>([])

  // update whenever cache entry is modified or the connection changes
  const updateLocal = () => {
    const id = connRef?.value?.id
    nodes.value = id ? treeCache[id] || [] : []
  }

  if (connRef) {
    watch(connRef, async (conn) => {
      if (conn && typeof conn === 'object') {
        await load(conn as { id: string, driver_type: string })
      }
      updateLocal()
    }, { immediate: true })
  }

  async function load(conn: { id: string, driver_type: string }) {
    if (!conn || !conn.id)
      return
    const id = conn.id
    if (treeCache[id])
      return // already fetched
    try {
      const cred = await GetCredential(id)
      const params: Record<string, any> = {}
      if (cred)
        params.credential_blob = cred
      const resp = await GetConnectionTree(conn.driver_type, params)
      treeCache[id] = normalizeNodes(resp.nodes || [])
      console.debug('useConnectionTree.load: cached nodes for', id, treeCache[id])
    }
    catch (err) {
      console.error('useConnectionTree load', id, err)
      treeCache[id] = []
    }
    updateLocal()
  }

  function getTableNames(): string[] {
    const out: string[] = []
    const gather = (items: any[]) => {
      for (const n of items) {
        // log each node inspected
        console.log('getTableNames inspecting', n.label, 'type', n.node_type)
        if (['table', 'view', 'collection'].includes(n.node_type)) {
          out.push(n.label)
        }
        if (Array.isArray(n.children))
          gather(n.children)
      }
    }
    gather(nodes.value)
    console.log('getTableNames result', out)
    return out
  }

  function getColumns(tableName: string): string[] {
    let cols: string[] = []
    const findTable = (items: any[]) => {
      for (const n of items) {
        if (n.label === tableName && Array.isArray(n.children)) {
          cols = n.children.map((c: any) => c.label)
          return true
        }
        if (Array.isArray(n.children) && findTable(n.children))
          return true
      }
      return false
    }
    findTable(nodes.value)
    return cols
  }

  return {
    nodes,
    load,
    getTableNames,
    getColumns,
    cache: treeCache,
  }
}
