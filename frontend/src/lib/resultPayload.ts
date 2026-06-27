export type ResultViewType = 'rdbms' | 'document' | 'kv' | null

function unwrapPayloadEnvelope(value: unknown): unknown {
  if (!value || typeof value !== 'object')
    return value

  const record = value as Record<string, unknown>
  if ('Payload' in record)
    return record.Payload

  return value
}

function unwrapResultVariant(value: unknown): unknown {
  if (!value || typeof value !== 'object')
    return value

  const record = value as Record<string, unknown>
  if (record.sql)
    return record.sql
  if (record.Sql)
    return record.Sql
  if (record.document)
    return record.document
  if (record.Document)
    return record.Document
  if (record.kv)
    return record.kv
  if (record.Kv)
    return record.Kv

  return value
}

function normalizeTopLevelKeys(value: unknown): unknown {
  if (!value || typeof value !== 'object' || Array.isArray(value))
    return value

  const out: Record<string, unknown> = {}
  for (const [key, fieldValue] of Object.entries(value as Record<string, unknown>)) {
    const normalizedKey = key.charAt(0).toLowerCase() + key.slice(1)
    out[normalizedKey] = fieldValue
  }
  return out
}

export function unwrapExecPayload(result: unknown): unknown {
  let current = unwrapPayloadEnvelope(result || {})
  current = unwrapResultVariant(current)
  current = unwrapResultVariant(current)
  return normalizeTopLevelKeys(current)
}

export function resultViewType(payload: unknown): ResultViewType {
  if (!payload || typeof payload !== 'object')
    return null

  const record = payload as Record<string, unknown>
  if (record.columns)
    return 'rdbms'
  if (record.documents !== undefined)
    return 'document'
  if (record.data !== undefined)
    return 'kv'
  return null
}

export function isEmptyExecPayload(payload: unknown): boolean {
  if (payload == null)
    return true

  if (Array.isArray(payload))
    return payload.length === 0

  if (typeof payload !== 'object')
    return false

  const record = payload as Record<string, unknown>

  if (Array.isArray(record.columns) && Array.isArray(record.rows))
    return record.columns.length === 0 && record.rows.length === 0

  if (record.documents !== undefined) {
    if (Array.isArray(record.documents))
      return record.documents.length === 0
    return false
  }

  if (record.data !== undefined) {
    if (!record.data || typeof record.data !== 'object')
      return false
    return Object.keys(record.data as Record<string, unknown>).length === 0
  }

  return Object.keys(record).length === 0
}
