import { ref } from 'vue'

/**
 * Shared modal state and mutation helper for row editor modals used
 * across the three ResultViewer components (RDBMS, Document, KeyValue).
 *
 * @param {Function} [buildFilter] - optional function (row) => string that
 *   produces a default filter expression for the given row.
 */
export function useRowEditorModal(buildFilter) {
  const showEditor = ref(false)
  const editorOperation = ref('update')
  const editorRow = ref(null)
  const editorFilter = ref('')
  const editorSource = ref('')

  function openEditor(op, row) {
    editorOperation.value = op
    editorRow.value = row
    editorFilter.value = buildFilter ? buildFilter(row) : ''
    editorSource.value = ''
    showEditor.value = true
  }

  function closeEditor() {
    showEditor.value = false
  }

  async function performMutation(connection, params, onMutated) {
    const { mutateRow } = await import('@/composables/useRowMutation')
    try {
      await mutateRow(
        connection,
        params.operation === 'delete' ? 3 : 2,
        params.source,
        params.values || {},
        params.filter,
      )
      onMutated?.()
    }
    catch (err) {
      console.error('mutation failed', err)
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
