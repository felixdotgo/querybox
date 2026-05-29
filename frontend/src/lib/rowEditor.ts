import type { ColumnSchema, EditorFieldDescriptor, EditorFieldKind } from '@/lib/types'

const INTEGER_TYPE_RE = /\b(?:tinyint|smallint|mediumint|int|integer|bigint|serial|bigserial)\b/i
const DECIMAL_TYPE_RE = /\b(?:decimal|numeric|float|double|real)\b/i
const BOOLEAN_TYPE_RE = /\b(?:bool|boolean)\b/i
const JSON_TYPE_RE = /\b(?:json|jsonb|array|object|struct|map|record|variant)\b/i
const TEXTAREA_TYPE_RE = /\b(?:text|ntext|mediumtext|longtext|clob|xml)\b/i
const TEXT_LIKE_TYPE_RE = /\b(?:text|varchar|char|character|bpchar|citext|xml|clob)\b/i

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isNumericValue(value: unknown): boolean {
  return typeof value === 'number' && Number.isFinite(value)
}

function parseJsonContainerString(value: unknown): unknown {
  if (typeof value !== 'string')
    return null
  const trimmed = value.trim()
  if (!(trimmed.startsWith('{') || trimmed.startsWith('[')))
    return null
  try {
    const parsed = JSON.parse(trimmed)
    return Array.isArray(parsed) || isPlainObject(parsed) ? parsed : null
  }
  catch {
    return null
  }
}

function kindFromDbType(rawType = ''): { kind: EditorFieldKind, numericMode?: 'integer' | 'decimal' } {
  const type = rawType.trim().toLowerCase()
  if (JSON_TYPE_RE.test(type))
    return { kind: 'json' }
  if (BOOLEAN_TYPE_RE.test(type))
    return { kind: 'boolean' }
  if (INTEGER_TYPE_RE.test(type))
    return { kind: 'number', numericMode: 'integer' }
  if (DECIMAL_TYPE_RE.test(type))
    return { kind: 'number', numericMode: 'decimal' }
  if (TEXTAREA_TYPE_RE.test(type))
    return { kind: 'textarea' }
  return { kind: 'text' }
}

function kindFromValue(value: unknown): { kind: EditorFieldKind, numericMode?: 'integer' | 'decimal' } {
  if (Array.isArray(value) || isPlainObject(value))
    return { kind: 'json' }
  if (parseJsonContainerString(value))
    return { kind: 'json' }
  if (typeof value === 'boolean')
    return { kind: 'boolean' }
  if (isNumericValue(value))
    return { kind: 'number', numericMode: Number.isInteger(value) ? 'integer' : 'decimal' }
  if (typeof value === 'string' && value.length > 120)
    return { kind: 'textarea' }
  return { kind: 'text' }
}

export function buildEditorFieldsFromSchema(
  row: Record<string, unknown>,
  columns: ColumnSchema[] = [],
): EditorFieldDescriptor[] {
  return Object.entries(row).map(([key, value]) => {
    const meta = columns.find(column => column.name === key)
    const fromValue = kindFromValue(value)
    const fromSchema = meta ? kindFromDbType(meta.type) : null
    const hasJsonStringValue = !!parseJsonContainerString(value)
    const isTextLikeSchema = !!meta?.type && TEXT_LIKE_TYPE_RE.test(meta.type)
    const inferred = (isTextLikeSchema && fromValue.kind === 'json') || (!fromSchema && fromValue.kind === 'json')
      ? fromValue
      : (fromSchema ?? fromValue)
    const serializeJsonAsString = inferred.kind === 'json'
      && isTextLikeSchema
      && (hasJsonStringValue || Array.isArray(value) || isPlainObject(value))
    return {
      key,
      label: key,
      kind: inferred.kind,
      nullable: meta?.nullable ?? true,
      rawType: meta?.type,
      numericMode: inferred.numericMode,
      serializeJsonAsString,
      value,
    }
  })
}

export function buildEditorFieldsFromValue(row: Record<string, unknown>): EditorFieldDescriptor[] {
  return Object.entries(row).map(([key, value]) => {
    const inferred = kindFromValue(value)
    return {
      key,
      label: key,
      kind: inferred.kind,
      numericMode: inferred.numericMode,
      nullable: value === null,
      serializeJsonAsString: inferred.kind === 'json' && typeof value === 'string' && !!parseJsonContainerString(value),
      value,
    }
  })
}

export function getEditorField(fields: EditorFieldDescriptor[], key: string): EditorFieldDescriptor | undefined {
  return fields.find(field => field.key === key)
}

export function isJsonEditorField(field?: EditorFieldDescriptor): boolean {
  return field?.kind === 'json'
}

export function coerceEditorValue(field: EditorFieldDescriptor, value: unknown): unknown {
  if (value === null || value === undefined)
    return value

  if (field.kind === 'number') {
    if (typeof value === 'number')
      return Number.isFinite(value) ? value : null
    if (typeof value === 'string') {
      const trimmed = value.trim()
      if (!trimmed)
        return null
      const parsed = Number(trimmed)
      return Number.isFinite(parsed) ? parsed : null
    }
    return null
  }

  if (field.kind === 'boolean') {
    if (typeof value === 'boolean')
      return value
    if (typeof value === 'string')
      return value.toLowerCase() === 'true'
    if (typeof value === 'number')
      return value !== 0
  }

  if (field.kind === 'json' && typeof value === 'string') {
    try {
      return JSON.parse(value)
    }
    catch {
      return value
    }
  }

  return value
}
