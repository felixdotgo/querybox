import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getCredentialMock, mutateRowBindingMock } = vi.hoisted(() => ({
  getCredentialMock: vi.fn(),
  mutateRowBindingMock: vi.fn(),
}))

vi.mock('@/bindings/github.com/felixdotgo/querybox/services/connectionservice', () => ({
  GetCredential: getCredentialMock,
}))

vi.mock('@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager', () => ({
  MutateRow: mutateRowBindingMock,
}))

const { mutateRow } = await import('./useRowMutation')

describe('mutateRow', () => {
  beforeEach(() => {
    getCredentialMock.mockReset()
    mutateRowBindingMock.mockReset()
  })

  it('forwards the explicit database instead of deriving one from a schema-qualified PostgreSQL source', async () => {
    getCredentialMock.mockResolvedValue('cred-blob')
    mutateRowBindingMock.mockResolvedValue({ success: true })

    await mutateRow(
      { id: 'conn-1', driver_type: 'postgresql' },
      2,
      'public.users',
      { name: 'alice' },
      `id = '1'`,
      { id: 1 },
      'appdb',
    )

    expect(mutateRowBindingMock).toHaveBeenCalledWith(
      'postgresql',
      { credential_blob: 'cred-blob', database: 'appdb' },
      2,
      'public.users',
      { name: 'alice' },
      `id = '1'`,
      { id: 1 },
    )
  })

  it('forwards typed values and filter values unchanged', async () => {
    getCredentialMock.mockResolvedValue('cred-blob')
    mutateRowBindingMock.mockResolvedValue({ success: true })

    await mutateRow(
      { id: 'conn-1', driver_type: 'sqlite' },
      2,
      'users',
      { active: true, age: 25, meta: { role: 'admin' }, name: null },
      '',
      { id: 1 },
    )

    expect(mutateRowBindingMock).toHaveBeenCalledWith(
      'sqlite',
      { credential_blob: 'cred-blob' },
      2,
      'users',
      { active: true, age: 25, meta: { role: 'admin' }, name: null },
      '',
      { id: 1 },
    )
  })
})
