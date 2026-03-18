import type { Ref } from 'vue'
import { computed, ref } from 'vue'
import { GetCredential } from '@/bindings/github.com/felixdotgo/querybox/services/connectionservice'
import { GetCompletionFields } from '@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager'
import { useConnectionTree } from '@/composables/useConnectionTree'

// Module-level cache: connId:collection -> FieldInfo[] so results survive
// tab switches and aren't refetched for every editor keystroke.
const completionFieldsCache = new Map<string, Array<{ name: string, type: string }>>()

/**
 * Per-tab completion data composable.
 *
 * Wraps `useConnectionTree` and adds:
 *  - `getCompletionFields(collection)` for schemaless DBs
 *  - `primaryTable` — inferred from the active tab's selected node
 *  - smart-ranked suggestion helpers
 *
 * @param {import('vue').Ref} tabRef - reactive reference to the current tab object
 */
export function useTabCompletion(tabRef: Ref<any>) {
  const { nodes, load, getTableNames, getColumns, getColumnDetails, getSchema, getAllSchemas } = useConnectionTree()

  // Lazily-fetched sampled fields, keyed by collection name
  const completionFieldsLoading = ref(false)

  /**
   * The primary table/collection for the active tab — inferred from the
   * selected tree node in the tab context.  Used to rank suggestions.
   */
  const primaryTable = computed(() => {
    const node = tabRef?.value?.context?.node
    if (!node)
      return null
    if (['table', 'view', 'collection'].includes(node.node_type)) {
      return node.label
    }
    return null
  })

  /**
   * The connection object for the active tab.
   */
  const conn = computed(() => tabRef?.value?.context?.conn || null)

  /**
   * The driver type string (e.g. 'mongodb', 'postgresql').
   */
  const driverType = computed(() => conn.value?.driver_type || '')

  /**
   * Whether this DB is schemaless (favouring GetCompletionFields over DescribeSchema).
   */
  const isSchemaless = computed(() => {
    return ['mongodb', 'arangodb', 'redis'].includes(driverType.value)
  })

  /**
   * Cache key prefix scoped to the connection.
   */
  function cacheKey(collection: string) {
    const id = conn.value?.id || 'unknown'
    return `${id}:${collection}`
  }

  /**
   * Return cached sampled fields for `collection`, or fetch them on first call.
   * Returns an array of { name, type? } objects.
   * @param {string} collection
   * @returns {Promise<Array<{name:string, type?:string}>>} cached or fetched completion fields
   */
  async function getCompletionFields(collection: string): Promise<Array<{ name: string, type: string }>> {
    if (!collection)
      return []
    const key = cacheKey(collection)
    if (completionFieldsCache.has(key))
      return completionFieldsCache.get(key)!

    const connection = conn.value
    if (!connection)
      return []

    completionFieldsLoading.value = true
    try {
      const cred = await GetCredential(connection.id)
      const params: Record<string, string> = {}
      if (cred)
        params.credential_blob = cred

      // Determine database from tree: try to find the database ancestor of the
      // selected collection node, or fall back to empty string.
      const database = inferDatabase(collection)

      const resp = await GetCompletionFields(connection.driver_type, params, database, collection)
      const fields = (resp?.fields || []).map((f: any) => ({ name: f.name, type: f.type || '' }))
      completionFieldsCache.set(key, fields)
      return fields
    }
    catch {
      completionFieldsCache.set(key, [])
      return []
    }
    finally {
      completionFieldsLoading.value = false
    }
  }

  /**
   * Attempt to resolve the database name from the connection tree nodes.
   * Walks the tree looking for a DATABASE ancestor of the named collection.
   * @param {string} collectionName
   * @returns {string} database name or empty string if not found
   */
  function inferDatabase(collectionName: string): string {
    function search(items: any[], parent: string): string | null {
      for (const n of items) {
        if (n.label === collectionName)
          return parent
        if (Array.isArray(n.children)) {
          const found = search(n.children, n.node_type === 'database' ? n.label : parent)
          if (found !== null)
            return found
        }
      }
      return null
    }
    return search(nodes.value, '') || ''
  }

  /**
   * Eagerly prefetch completion fields for the primary collection when the
   * tab becomes active and the DB is schemaless.  Call this from the editor
   * mount or tab-activated hook.
   */
  async function prefetchForPrimaryTable() {
    if (!isSchemaless.value)
      return
    const table = primaryTable.value
    if (table) {
      await getCompletionFields(table)
    }
  }

  /**
   * Invalidate cached completion data for the current connection.
   * Called when the user reconnects or refreshes the schema tree.
   */
  function invalidate() {
    const id = conn.value?.id
    if (!id)
      return
    for (const key of completionFieldsCache.keys()) {
      if (key.startsWith(`${id}:`))
        completionFieldsCache.delete(key)
    }
  }

  return {
    // Re-exports from useConnectionTree
    nodes,
    load,
    getTableNames,
    getColumns,
    getColumnDetails,
    getSchema,
    getAllSchemas,
    // New
    primaryTable,
    conn,
    driverType,
    isSchemaless,
    completionFieldsLoading,
    getCompletionFields,
    prefetchForPrimaryTable,
    invalidate,
  }
}
