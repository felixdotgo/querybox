import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'

const { getCredentialMock, execPluginMock } = vi.hoisted(() => ({
  getCredentialMock: vi.fn(),
  execPluginMock: vi.fn(),
}))

vi.mock('@/bindings/github.com/felixdotgo/querybox/services/connectionservice', () => ({
  GetCredential: getCredentialMock,
}))

vi.mock('@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager', () => ({
  ExecPlugin: execPluginMock,
}))

const { useResultSort } = await import('./useResultSort')

describe('useResultSort', () => {
  beforeEach(() => {
    getCredentialMock.mockReset()
    execPluginMock.mockReset()
  })

  it('uses the explicit database from tab context instead of deriving from schema names', async () => {
    getCredentialMock.mockResolvedValue('cred-blob')
    execPluginMock.mockResolvedValue({
      result: {
        Payload: {
          Sql: {
            columns: [],
            rows: [],
          },
        },
      },
    })

    const query = ref('SELECT * FROM public.users')
    const connection = ref({ id: 'conn-1', driver_type: 'postgresql' })
    const database = ref('appdb')

    const { handleSorterChange } = useResultSort({ query, connection, database })
    await handleSorterChange({ columnKey: 'id', order: 'ascend' })

    expect(execPluginMock).toHaveBeenCalledWith(
      'postgresql',
      { credential_blob: 'cred-blob', database: 'appdb' },
      'SELECT * FROM public.users',
      { 'sort-column': 'id', 'sort-direction': 'asc' },
    )
  })
})
