import { describe, expect, it } from 'vitest'
import { resultViewType, unwrapExecPayload } from './resultPayload'

describe('resultPayload', () => {
  it('unwraps SQL payloads from protobuf envelopes', () => {
    const payload = unwrapExecPayload({
      Payload: {
        Sql: {
          columns: ['id'],
          rows: [[1]],
        },
      },
    })

    expect(payload).toEqual({ columns: ['id'], rows: [[1]] })
    expect(resultViewType(payload)).toBe('rdbms')
  })

  it('routes Redis key/value responses to the key-value viewer', () => {
    const payload = unwrapExecPayload({
      Kv: {
        data: {
          key: 'session:1',
          value: 'hello',
        },
      },
    })

    expect(payload).toEqual({ data: { key: 'session:1', value: 'hello' } })
    expect(resultViewType(payload)).toBe('kv')
  })

  it('routes Redis collection and scan responses to the document viewer', () => {
    const payload = unwrapExecPayload({
      document: {
        documents: [
          { key: 'a' },
          { key: 'b' },
        ],
      },
    })

    expect(payload).toEqual({ documents: [{ key: 'a' }, { key: 'b' }] })
    expect(resultViewType(payload)).toBe('document')
  })
})
