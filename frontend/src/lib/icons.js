/**
 * Central icon registry for QueryBox.
 *
 * Rules:
 *  - Always import icons FROM this file; never import @vicons/ionicons5 directly
 *    in components.  Swapping the icon library means editing only this file.
 *  - Use <n-icon> (Naive UI) as the wrapper in templates.
 *  - Render-function usage: h(NIcon, { size }, { default: () => h(IconComponent) })
 *  - Preferred sizes: 16 for toolbar / buttons, 14 for tree-node prefixes.
 */
import {
  // actions (filled variants aliased to keep existing export names unchanged)
  AddCircle,
  Analytics,
  ArrowDown,
  Cash,
  ChevronDown,
  CodeSlash,
  CreateOutline,
  Documents,
  EllipsisHorizontal,
  Eye,
  Flash,
  Folder,
  Grid,
  Key,
  Pencil,
  // tree / navigation
  Layers,
  Library,
  Pin,
  Play,
  Refresh,
  Search,
  Server,
  Terminal,
  Time,
  Trash,
} from '@vicons/ionicons5'

export {
  AddCircle, // new connection toolbar button
  Analytics, // explain query button
  ArrowDown, // log panel auto-scroll toggle
  Cash, // cost / dollar
  ChevronDown, // footer collapse toggle (rotate -90deg when collapsed)
  CodeSlash, // node_type === "column"
  CreateOutline, // edit connection
  Documents, // unknown / generic fallback
  EllipsisHorizontal, // three-dot context menu trigger
  Eye, // "select" action on tree nodes
  Flash, // "Connect" action / execution time (bolt)
  Folder, // node_type === "group" (category folder)
  Grid, // node_type === "table"
  Key, // primary key indicator
  Layers, // driver group node
  Library, // node_type === "database"
  Pin, // pinned column indicator (filled)
  Play, // execute query button
  Refresh, // "Refresh" action on connection row
  Search, // filter input prefix
  Server, // connection node / rows (databases)
  Terminal, // logs panel header
  Time, // planning time (clock)
  Trash, // "Delete" action on connection row
  Pencil, // generic edit/pencil icon for row‑mutation, etc.
}

// colours used to style datatype badges in result viewer headers
// each value should be a CSS color or custom property
export const dataTypeColorMap = {
  int: 'var(--n-info-color, #3b82f6)',
  float: 'var(--n-info-color, #3b82f6)',
  double: 'var(--n-info-color, #3b82f6)',
  decimal: 'var(--n-info-color, #3b82f6)',
  numeric: 'var(--n-info-color, #3b82f6)',
  real: 'var(--n-info-color, #3b82f6)',
  bool: 'var(--n-success-color, #18a058)',
  boolean: 'var(--n-success-color, #18a058)',
  date: 'var(--n-warning-color, #f59e0b)',
  time: 'var(--n-warning-color, #f59e0b)',
  timestamp: 'var(--n-warning-color, #f59e0b)',
  datetime: 'var(--n-warning-color, #f59e0b)',
  char: 'var(--n-neutral-color, #6b7280)',
  text: 'var(--n-neutral-color, #6b7280)',
  clob: 'var(--n-neutral-color, #6b7280)',
  blob: 'var(--n-neutral-color, #6b7280)',
  binary: 'var(--n-neutral-color, #6b7280)',
  varbinary: 'var(--n-neutral-color, #6b7280)',
}

/**
 * Return a CSS color string appropriate for the given SQL type.
 * Falls back to neutral if no mapping matches.
 *
 * @param {string} type
 * @returns {string} CSS color value from dataTypeColorMap or default if no match
 */
export function getDataTypeColor(type) {
  if (!type || typeof type !== 'string')
    return dataTypeColorMap.text
  const t = type.toLowerCase()
  for (const key of Object.keys(dataTypeColorMap)) {
    if (t.includes(key)) {
      return dataTypeColorMap[key]
    }
  }
  return dataTypeColorMap.text
}

/**
 * Maps node_type strings (as returned by plugins via the proto node_type field)
 * to their icon component.  Falls back to nodeTypeFallbackIcon when unknown.
 * @type {Record<string, object>}
 */
export const nodeTypeIconMap = {
  server: Server,
  database: Server,
  schema: Layers,
  group: Folder,
  table: Grid,
  column: CodeSlash,
  action: AddCircle,
}

/** Used when a plugin node has no recognised node_type value. */
export const nodeTypeFallbackIcon = Documents

/**
 * Maps ConnectionTreeAction type strings (constants defined in pkg/plugin/plugin.go)
 * to their icon component.  Falls back to actionTypeFallbackIcon when unknown.
 * @type {Record<string, object>}
 */
export const actionTypeIconMap = {
  'select': Eye,
  'describe': CodeSlash,
  'create-database': AddCircle,
  'create-table': AddCircle,
  'drop-database': Trash,
  'drop-table': Trash,
}

/** Used when an action has no recognised type value. */
export const actionTypeFallbackIcon = Flash
