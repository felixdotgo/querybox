import { describe, expect, it } from 'vitest'
import { isEmptyExecPayload, resultViewType, unwrapExecPayload } from './resultPayload'

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

  it('normalizes capitalized protobuf fields after unwrapping', () => {
    const payload = unwrapExecPayload({
      Payload: {
        Document: {
          Documents: [
            { key: 'session:1' },
          ],
        },
      },
    })

    expect(payload).toEqual({ documents: [{ key: 'session:1' }] })
    expect(resultViewType(payload)).toBe('document')
  })

  it('treats empty typed payloads as empty results', () => {
    expect(isEmptyExecPayload({ columns: [], rows: [] })).toBe(true)
    expect(isEmptyExecPayload({ documents: [] })).toBe(true)
    expect(isEmptyExecPayload({ data: {} })).toBe(true)
    expect(isEmptyExecPayload({})).toBe(true)
    expect(isEmptyExecPayload(null)).toBe(true)
  })

  it('keeps unsupported but non-empty payloads visible for fallback rendering', () => {
    expect(isEmptyExecPayload({ summary: 'ok', count: 1 })).toBe(false)
    expect(resultViewType({ summary: 'ok', count: 1 })).toBe(null)
  })
})
