/**
 * Utilities for parsing hierarchical connection tree node keys.
 *
 * Node keys follow the format:
 *   <connId>:<database>:<schema>:<qualified.table>
 *
 * Not all segments are always present — flat-hierarchy drivers (e.g. SQLite)
 * may only have <connId>:<table>.
 */

/** Strip the `<connId>:` prefix from a node key if present. */
export function stripConnPrefix(connId: string, key: string): string {
  if (!key) return key
  const prefix = `${connId}:`
  return key.startsWith(prefix) ? key.slice(prefix.length) : key
}

/**
 * Extract the table name from a node key.
 *
 * After stripping the connection prefix, returns the last colon-separated
 * segment (which is the table/collection name, possibly schema-qualified
 * like "public.users").
 *
 * Returns `null` if no valid table name can be extracted.
 */
export function extractTableName(connId: string | undefined, key: string | undefined): string | null {
  if (!key || typeof key !== 'string') return null
  let k = connId ? stripConnPrefix(connId, key) : key
  const lastColon = k.lastIndexOf(':')
  if (lastColon !== -1) {
    k = k.slice(lastColon + 1)
  }
  return k || null
}

/**
 * Extract the database name from a node key.
 *
 * After stripping the connection prefix, returns the characters before the
 * first `.` or `:` separator — this represents the database name for most
 * drivers. Returns `null` for flat-hierarchy drivers where no separator
 * exists.
 */
export function extractDatabase(connId: string | undefined, key: string | undefined): string | null {
  if (!key || typeof key !== 'string') return null
  let k = connId ? stripConnPrefix(connId, key) : key

  const dot = k.indexOf('.')
  const col = k.indexOf(':')
  let cut = -1
  if (dot !== -1 && col !== -1) {
    cut = Math.min(dot, col)
  } else if (dot !== -1) {
    cut = dot
  } else if (col !== -1) {
    cut = col
  }

  if (cut !== -1) {
    const db = k.slice(0, cut)
    return db || null
  }

  return null
}
