import type { Ref } from 'vue'
import { reactive, ref, watch } from 'vue'
import { GetCredential } from '@/bindings/github.com/felixdotgo/querybox/services/connectionservice'
import { DescribeSchema, GetConnectionTree } from '@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager'

// global reactive cache mapping connection id -> nodes array
const treeCache: Record<string, any[]> = reactive({})

// schemaCache maps connection id -> tableName -> Schema object returned by plugin
// TODO: persist these entries on disk (via the connection service or similar)
// so that table information can survive restarts and be available offline.
const schemaCache: Record<string, Record<string, any>> = reactive({})

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
      // load schema info in parallel; ignore errors
      try {
        // @ts-expect-error: may be generated later
        const schemaResp = await DescribeSchema(conn.driver_type, params)
        console.debug('useConnectionTree.load: raw schema response', id, schemaResp)
        console.debug('useConnectionTree.load: tables count', id, schemaResp?.tables?.length)
        const tableMap: Record<string, any> = {}
        if (schemaResp && Array.isArray(schemaResp.tables)) {
          for (const t of schemaResp.tables) {
            if (t && t.name) {
              // cache under the exact name returned by the plugin
              tableMap[t.name] = t
              // if the name contains a dot, also cache the suffix after the
              // first segment. many plugins return qualified names such as
              // "public.users"; workspace tabs, however, strip the leading
              // database/schema when constructing the key. duplicating the
              // entry here keeps lookups simple and backwards-compatible.
              const idx = t.name.indexOf('.')
              if (idx !== -1 && idx < t.name.length - 1) {
                const suffix = t.name.slice(idx + 1)
                if (!(suffix in tableMap)) {
                  tableMap[suffix] = t
                }
              }
            }
          }
        }
        schemaCache[id] = tableMap
        console.debug('useConnectionTree.load: cached schema for', id, tableMap)
      }
      catch (err) {
        console.error('useConnectionTree.load schema error', id, err)
        schemaCache[id] = {}
      }
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

  /**
   * Return rich column metadata for `tableName` from the schema cache.
   * Each item includes { name, type, nullable, primary_key }.
   * Falls back to plain name-only objects when schema data is unavailable.
   */
  function getColumnDetails(tableName: string): Array<{ name: string, type: string, nullable: boolean, primary_key: boolean }> {
    const schema = getSchema(tableName)
    if (schema && Array.isArray(schema.columns) && schema.columns.length > 0) {
      return schema.columns.map((c: any) => ({
        name: c.name || '',
        type: c.type || '',
        nullable: !!c.nullable,
        primary_key: !!c.primary_key,
      }))
    }
    // Fall back to tree-children names when schema cache has no columns metadata
    return getColumns(tableName).map(name => ({ name, type: '', nullable: true, primary_key: false }))
  }

  function getSchema(tableName: string): any | null {
    const id = connRef?.value?.id
    if (!id)
      return null
    if (!schemaCache[id])
      return null

    // first try the direct key
    let result = schemaCache[id][tableName]
    if (result)
      return result

    // if the key contains a dot we may have cached the suffix; try that
    const idx = tableName.indexOf('.')
    if (idx !== -1 && idx < tableName.length - 1) {
      const suffix = tableName.slice(idx + 1)
      result = schemaCache[id][suffix]
      if (result)
        return result
    }

    // as a last resort, search for any entry whose name ends with ".<tableName>"
    // this is slightly more expensive but guards against deeper qualifiers
    for (const k in schemaCache[id]) {
      if (k.endsWith(`.${tableName}`)) {
        return schemaCache[id][k]
      }
    }

    return null
  }

  function getAllSchemas(): Record<string, any> {
    const id = connRef?.value?.id
    if (!id)
      return {}
    return schemaCache[id] || {}
  }

  // fetch schema metadata for the specified table only and merge the results
  // into the cache.  called lazily when the user selects a table that hasn't
  // been previously described.
  async function fetchSchema(table?: string) {
    const id = connRef?.value?.id
    const conn = connRef?.value
    if (!id || !conn)
      return
    const cred = await GetCredential(id)
    const params: Record<string, any> = {}
    if (cred)
      params.credential_blob = cred

    // split a qualified table name into database and table filters
    let dbFilter = ''
    let tblFilter = ''
    if (table) {
      const parts = table.split('.')
      if (parts.length > 1) {
        dbFilter = parts[0]
        tblFilter = parts.slice(1).join('.')
      }
      else {
        tblFilter = table
      }
    }

    try {
      console.debug('useConnectionTree.fetchSchema', id, 'table', table, 'dbFilter', dbFilter, 'tblFilter', tblFilter)
      // @ts-expect-error: may be generated later
      const schemaResp = await DescribeSchema(conn.driver_type, params, dbFilter, tblFilter)
      console.debug('useConnectionTree.fetchSchema: raw response', id, table, schemaResp)
      const tableMap: Record<string, any> = {}
      if (schemaResp && Array.isArray(schemaResp.tables)) {
        for (const t of schemaResp.tables) {
          if (t && t.name) {
            tableMap[t.name] = t
            const idx = t.name.indexOf('.')
            if (idx !== -1 && idx < t.name.length - 1) {
              const suffix = t.name.slice(idx + 1)
              if (!(suffix in tableMap)) {
                tableMap[suffix] = t
              }
            }
          }
        }
        // merge into existing cache rather than clobber
        schemaCache[id] = { ...(schemaCache[id] || {}), ...tableMap }
        console.debug('useConnectionTree.fetchSchema: merged schema for', id, table, tableMap)
      }
    }
    catch (err) {
      console.error('useConnectionTree.fetchSchema error', id, table, err)
    }
  }

  return {
    nodes,
    load,
    getTableNames,
    getColumns,
    getColumnDetails,
    getSchema,
    getAllSchemas,
    fetchSchema,
    cache: treeCache,
    schemaCache,
  }
}
