import { useNotification } from 'naive-ui'
import { ref } from 'vue'

/**
 * Shared modal state and mutation helper for row editor modals used
 * across the three ResultViewer components (RDBMS, Document, KeyValue).
 *
 * @param {Function} [buildFilter] - optional function (row) => string that
 *   produces a default filter expression for the given row.
 */
export function useRowEditorModal(buildFilter) {
  const notification = useNotification()
  const showEditor = ref(false)
  const editorOperation = ref('update')
  const editorRow = ref(null)
  const editorFilter = ref('')
  const editorSource = ref('')

  function openEditor(op, row, source = '', filter = null) {
    editorOperation.value = op
    editorRow.value = row
    editorFilter.value = filter !== null ? filter : (buildFilter ? buildFilter(row) : '')
    editorSource.value = source
    showEditor.value = true
  }

  function closeEditor() {
    showEditor.value = false
  }

  async function performMutation(connection, params, onMutated) {
    const { mutateRow } = await import('@/composables/useRowMutation')
    try {
      const res = await mutateRow(
        connection,
        params.operation === 'delete' ? 3 : 2,
        params.source,
        params.values || {},
        params.filter,
      )
      if (res && (res.success === false || res.error)) {
        const msg = res.error || 'Operation failed'
        console.error('mutation failed', msg)
        notification.error({ title: 'Mutation failed', content: msg, duration: 5000 })
        return
      }
      const opLabel = params.operation === 'delete' ? 'Row deleted' : 'Row updated'
      notification.success({ title: opLabel, duration: 3000 })
      onMutated?.({ operation: params.operation, source: params.source, filter: params.filter })
    }
    catch (err) {
      console.error('mutation failed', err)
      notification.error({ title: 'Mutation failed', content: err?.message || String(err), duration: 5000 })
    }
  }

  return {
    showEditor,
    editorOperation,
    editorRow,
    editorFilter,
    editorSource,
    openEditor,
    closeEditor,
    performMutation,
  }
}
