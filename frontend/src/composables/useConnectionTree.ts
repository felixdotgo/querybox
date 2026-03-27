import type { Ref } from 'vue'
import { reactive, ref, watch } from 'vue'
import { GetCredential } from '@/bindings/github.com/felixdotgo/querybox/services/connectionservice'
import { DescribeSchema, GetConnectionTree } from '@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager'
import type { ColumnSchema, Connection, TableSchema, TreeNode } from '@/lib/types'

// global reactive cache mapping connection id -> nodes array
const treeCache: Record<string, TreeNode[]> = reactive({})

// schemaCache maps connection id -> tableName -> Schema object returned by plugin
// TODO: persist these entries on disk (via the connection service or similar)
// so that table information can survive restarts and be available offline.
const schemaCache: Record<string, Record<string, TableSchema>> = reactive({})

// map proto enum numbers to lowercase names; used both here and by the
// ConnectionsPanel component.  Exported so tests can verify tagging logic.
export const NODE_TYPE_ENUM_MAP: Record<number, string> = {
  1: 'database',
  2: 'table',
  3: 'column',
  4: 'schema',
  5: 'view',
  6: 'action',
  7: 'collection',
  8: 'key',
  9: 'group',
}

// Recursively tag every node with its owning connection id, normalize the
// `node_type` value, and sort/uniq siblings.  This mirrors nearly identical
// logic previously in ConnectionsPanel.tagWithConnId; moving it here makes it
// easier to test and avoids regenerating fresh objects on every render (which
// confused the tree view and led to duplicate entries when a node was
// expanded/collapsed repeatedly).
export function tagWithConnId(nodes: TreeNode[], connId: string, _prefix?: string): TreeNode[] {
  // `_prefix` accumulates the full ancestor key path so that sibling nodes
  // in different databases (e.g. both have a "public" schema) receive distinct
  // keys, preventing NaiveUI from conflating their expansion state.
  const prefix = _prefix !== undefined ? _prefix : connId
  const seenKeys = new Set<string>()

  const tagged = nodes.map((n) => {
    const nodeType = typeof n.node_type === 'number'
      ? (NODE_TYPE_ENUM_MAP[n.node_type] ?? String(n.node_type))
      : n.node_type

    const base: TreeNode = {
      ...n,
      key: `${prefix}:${n.key}`,
      _connectionId: connId,
      node_type: nodeType,
    }
    if (n.children) {
      base.children = tagWithConnId(n.children, connId, base.key)
    }
    return base
  })

  // deduplicate siblings by key preserving original order
  const out: TreeNode[] = []
  for (const t of tagged) {
    if (!seenKeys.has(t.key)) {
      seenKeys.add(t.key)
      out.push(t)
    }
  }

  // stable sort: action nodes first, then alphabetic by label
  out.sort((a, b) => {
    const aIsAction = a.node_type === 'action'
    const bIsAction = b.node_type === 'action'
    if (aIsAction && !bIsAction)
      return -1
    if (!aIsAction && bIsAction)
      return 1
    return (a.label ?? '').localeCompare(b.label ?? '')
  })

  return out
}

function normalizeNodes(nodes: TreeNode[]): TreeNode[] {
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
export function useConnectionTree(connRef?: Ref<Connection | null>) {
  const nodes = ref<TreeNode[]>([])

  // update whenever cache entry is modified or the connection changes
  const updateLocal = () => {
    const id = connRef?.value?.id
    nodes.value = id ? treeCache[id] || [] : []
  }

  if (connRef) {
    watch(connRef, async (conn) => {
      if (conn && typeof conn === 'object') {
        await load(conn)
      }
      updateLocal()
    }, { immediate: true })
  }

  async function load(conn: Pick<Connection, 'id' | 'driver_type'>) {
    if (!conn || !conn.id)
      return
    const id = conn.id
    if (treeCache[id])
      return // already fetched
    try {
      const cred = await GetCredential(id)
      const params: Record<string, string> = {}
      if (cred)
        params.credential_blob = cred
      const resp = await GetConnectionTree(conn.driver_type, params)
      treeCache[id] = normalizeNodes((resp?.nodes ?? []).filter(n => n !== null) as unknown as TreeNode[])
      // load schema info in parallel; ignore errors
      try {
        // @ts-expect-error: may be generated later
        const schemaResp = await DescribeSchema(conn.driver_type, params)
        const tableMap: Record<string, TableSchema> = {}
        if (schemaResp && Array.isArray(schemaResp.tables)) {
          for (const t of schemaResp.tables as TableSchema[]) {
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
      }
      catch (err) {
        console.error('useConnectionTree.load schema error', id, err)
        schemaCache[id] = {}
      }
    }
    catch (err) {
      console.error('useConnectionTree load', id, err)
      throw err
    }
    finally {
      updateLocal()
    }
  }

  function getTableNames(): string[] {
    const out: string[] = []
    const gather = (items: TreeNode[]) => {
      for (const n of items) {
        if (['table', 'view', 'collection'].includes(n.node_type as string)) {
          out.push(n.label)
        }
        if (Array.isArray(n.children))
          gather(n.children)
      }
    }
    gather(nodes.value)
    return out
  }

  function getColumns(tableName: string): string[] {
    let cols: string[] = []
    const findTable = (items: TreeNode[]) => {
      for (const n of items) {
        if (n.label === tableName && Array.isArray(n.children)) {
          cols = n.children.map(c => c.label)
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
  function getColumnDetails(tableName: string): ColumnSchema[] {
    const schema = getSchema(tableName)
    if (schema && Array.isArray(schema.columns) && schema.columns.length > 0) {
      return schema.columns.map((c: ColumnSchema) => ({
        name: c.name || '',
        type: c.type || '',
        nullable: !!c.nullable,
        primary_key: !!c.primary_key,
      }))
    }
    // Fall back to tree-children names when schema cache has no columns metadata
    return getColumns(tableName).map(name => ({ name, type: '', nullable: true, primary_key: false }))
  }

  function getSchema(tableName: string, overrideConn?: Pick<Connection, 'id' | 'driver_type'>): TableSchema | null {
    // connection id determined either from override or reactive ref
    const id = overrideConn?.id || connRef?.value?.id
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

  function getAllSchemas(): Record<string, TableSchema> {
    const id = connRef?.value?.id
    if (!id)
      return {}
    return schemaCache[id] || {}
  }

  // fetch schema metadata for the specified table only and merge the results
  // into the cache.  called lazily when the user selects a table that hasn't
  // been previously described.
  //
  // `database` is the actual database name to connect to (e.g. "mydb" for
  // PostgreSQL multi-DB setups).  When provided it is forwarded in params so
  // that buildConnString on the backend opens the correct database.  This is
  // distinct from the schema/dbFilter derived from the table name, which is
  // used as the DescribeSchema filter argument.
  async function fetchSchema(table?: string, overrideConn?: Pick<Connection, 'id' | 'driver_type'>, database?: string) {
    // choose connection info from override or reactive ref
    const conn = overrideConn || connRef?.value
    const id = conn?.id
    if (!id || !conn)
      return
    const cred = await GetCredential(id)
    const params: Record<string, string> = {}
    if (cred)
      params.credential_blob = cred
    // forward the actual database name so the backend DSN targets the right DB
    if (database)
      params.database = database

    // split a qualified table name into schema and table filters
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
      const schemaResp = await DescribeSchema(conn.driver_type, params, dbFilter, tblFilter)
      const tableMap: Record<string, TableSchema> = {}
      if (schemaResp && Array.isArray(schemaResp.tables)) {
        for (const t of schemaResp.tables as TableSchema[]) {
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
      }
    }
    catch (err) {
      console.error('useConnectionTree.fetchSchema error', id, table, err)
    }
  }

  // Note: `getSchema` and `fetchSchema` now accept an optional
  // `overrideConn` parameter allowing callers to specify a connection
  // object explicitly.  This is used by the workspace tabs so that
  // schema lookups follow the context of each tab rather than the
  // globally selected connection.
  return {
    nodes,
    load,
    getTableNames,
    getColumns,
    getColumnDetails,
    getSchema,
    getAllSchemas,
    fetchSchema,
    schemaCache,
    cache: treeCache,
  }
}
