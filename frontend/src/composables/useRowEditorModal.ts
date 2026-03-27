import { useNotification } from 'naive-ui'
import { ref, type Ref } from 'vue'
import type { MutationParams } from '@/lib/types'

/**
 * Shared modal state and mutation helper for row editor modals used
 * across the three ResultViewer components (RDBMS, Document, KeyValue).
 */
export function useRowEditorModal(buildFilter?: (row: Record<string, unknown>) => string) {
  const notification = useNotification()
  const showEditor: Ref<boolean> = ref(false)
  const editorOperation: Ref<string> = ref('update')
  const editorRow: Ref<Record<string, unknown> | null> = ref(null)
  const editorFilter: Ref<string> = ref('')
  const editorSource: Ref<string> = ref('')

  function openEditor(op: string, row: Record<string, unknown>, source = '', filter: string | null = null): void {
    editorOperation.value = op
    editorRow.value = row
    editorFilter.value = filter !== null ? filter : (buildFilter ? buildFilter(row) : '')
    editorSource.value = source
    showEditor.value = true
  }

  function closeEditor(): void {
    showEditor.value = false
  }

  async function performMutation(
    connection: { id: string; driver_type: string },
    params: MutationParams,
    onMutated?: (info: { operation: string; source: string; filter: string }) => void,
  ): Promise<void> {
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
    catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err)
      console.error('mutation failed', err)
      notification.error({ title: 'Mutation failed', content: message, duration: 5000 })
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
