import { beforeEach, describe, expect, it, vi } from 'vitest'
import { computed, ref } from 'vue'

const { execTreeActionMock, getCredentialMock, testConnectionMock } = vi.hoisted(() => ({
  execTreeActionMock: vi.fn(),
  getCredentialMock: vi.fn(),
  testConnectionMock: vi.fn(),
}))

vi.mock('naive-ui', () => ({
  useDialog: () => ({ error: vi.fn() }),
  useNotification: () => ({ error: vi.fn() }),
}))

vi.mock('@/bindings/github.com/felixdotgo/querybox/services/connectionservice', () => ({
  DeleteConnection: vi.fn(),
  GetCredential: getCredentialMock,
}))

vi.mock('@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager', () => ({
  ExecTreeAction: execTreeActionMock,
  TestConnection: testConnectionMock,
}))

const { useTreeActions } = await import('./useTreeActions')

function buildHarness() {
  const emit = vi.fn()
  const state = {
    connections: ref([{ id: 'redis-1', driver_type: 'redis', name: 'Local Redis' }]),
    connectionTrees: {},
    schemaCache: {},
    expandedKeys: ref([]),
    loadingNodes: ref({}),
    connecting: ref({}),
    selectedConnection: ref(null),
    pluginCaps: computed(() => ({ redis: ['query.execute'] })),
    loadResourceGraph: vi.fn(),
    emit,
  }

  return {
    actions: useTreeActions(state),
    emit,
    conn: state.connections.value[0],
  }
}

describe('useTreeActions', () => {
  beforeEach(() => {
    execTreeActionMock.mockReset()
    getCredentialMock.mockReset()
    testConnectionMock.mockReset()
    getCredentialMock.mockResolvedValue('cred-blob')
  })

  it('emits normalized Redis document payloads for result tabs', async () => {
    execTreeActionMock.mockResolvedValue({
      result: {
        Payload: {
          Document: {
            Documents: [
              { key: 'session:1' },
            ],
          },
        },
      },
    })

    const { actions, conn, emit } = buildHarness()

    await actions.runTreeAction(
      conn,
      { type: 'select', query: 'SCAN 0 COUNT 100', new_tab: true, title: 'Keys' },
      { key: 'redis-1/keys', node_type: 'keyspace', label: 'keys' },
    )

    expect(execTreeActionMock).toHaveBeenCalledWith(
      'redis',
      { credential_blob: 'cred-blob' },
      'SCAN 0 COUNT 100',
      {},
    )
    expect(emit).toHaveBeenCalledWith(
      'query-result',
      expect.objectContaining({
        result: { documents: [{ key: 'session:1' }] },
        error: null,
        tabKey: 'redis-1:redis-1/keys',
      }),
    )
  })

  it('emits concrete plugin errors into the result tab', async () => {
    execTreeActionMock.mockResolvedValue({
      error: 'NOAUTH Authentication required.',
    })

    const { actions, conn, emit } = buildHarness()

    await actions.runTreeAction(
      conn,
      { type: 'select', query: 'GET session:1', new_tab: true, title: 'session:1' },
      { key: 'redis-1/session:1', node_type: 'key', label: 'session:1' },
    )

    expect(emit).toHaveBeenCalledWith(
      'query-result',
      expect.objectContaining({
        result: null,
        error: 'NOAUTH Authentication required.',
        tabKey: 'redis-1:redis-1/session:1',
      }),
    )
  })
})
