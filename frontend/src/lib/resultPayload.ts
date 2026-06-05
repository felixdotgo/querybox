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

export function unwrapExecPayload(result: unknown): unknown {
  let current = unwrapPayloadEnvelope(result || {})
  current = unwrapResultVariant(current)
  current = unwrapResultVariant(current)
  return current
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
