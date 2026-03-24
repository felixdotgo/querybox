import { ref } from 'vue'
import { GetCredential } from '@/bindings/github.com/felixdotgo/querybox/services/connectionservice'
import { ExecPlugin } from '@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager'

type SortOrder = 'ascend' | 'descend' | false

interface SortState {
  columnKey: string | number
  order: SortOrder
}

interface Connection {
  id: string
  driver_type: string
}

interface UseResultSortOptions {
  query: { value: string }
  connection: { value: Connection | null | undefined }
  database?: { value: string | null | undefined }
}

export function useResultSort({ query, connection, database }: UseResultSortOptions) {
  // columnKey → current sort direction
  const sortStates = ref<Map<string, SortOrder>>(new Map())
  const isSorting = ref(false)
  // When set, overrides props.payload in the viewer (cleared on manual re-execute)
  const sortedPayload = ref<any>(null)

  async function handleSorterChange(state: SortState | SortState[] | null) {
    const single = Array.isArray(state) ? state[0] : state

    // null or order=false → user cleared sort; restore original payload
    if (!single || single.order === false) {
      sortStates.value = new Map()
      sortedPayload.value = null
      return
    }

    const colName = String(single.columnKey)
    const direction = single.order === 'ascend' ? 'asc' : 'desc'

    // update visual indicator — only active column, rest set to false
    const next = new Map<string, SortOrder>()
    next.set(colName, single.order)
    sortStates.value = next

    await executeSorted(colName, direction)
  }

  async function executeSorted(colName: string, direction: 'asc' | 'desc') {
    const conn = connection.value
    const q = query.value
    if (!conn || !q) return

    isSorting.value = true
    try {
      const connMap: Record<string, string> = {}
      const cred = await GetCredential(conn.id)
      if (cred) connMap.credential_blob = cred
      const db = database?.value
      if (db) connMap.database = db

      const result = await ExecPlugin(
        conn.driver_type,
        connMap,
        q,
        { 'sort-column': colName, 'sort-direction': direction },
      )

      // unwrap protobuf envelope — mirrors ResultViewer.vue payload computed
      let payload: any = result?.result?.Payload ?? result?.result ?? {}
      if (payload.Sql) payload = payload.Sql
      else if (payload.sql) payload = payload.sql

      sortedPayload.value = payload
    }
    catch (err) {
      console.error('sort re-execution failed', err)
      // reset on error — leave original payload visible
      sortStates.value = new Map()
      sortedPayload.value = null
    }
    finally {
      isSorting.value = false
    }
  }

  // Call when a new payload arrives (user manually re-executed the query)
  function resetSort() {
    sortStates.value = new Map()
    sortedPayload.value = null
    isSorting.value = false
  }

  return { sortStates, isSorting, sortedPayload, handleSorterChange, resetSort }
}
