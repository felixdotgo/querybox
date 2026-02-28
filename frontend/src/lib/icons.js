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
  Document,
  EllipsisHorizontal,
  Eye,
  Flash,
  Grid,
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
  Document, // unknown / generic fallback
  EllipsisHorizontal, // three-dot context menu trigger
  Eye, // "select" action on tree nodes
  Flash, // "Connect" action / execution time (bolt)
  Grid, // node_type === "table"
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
}

/**
 * Maps node_type strings (as returned by plugins via the proto node_type field)
 * to their icon component.  Falls back to nodeTypeFallbackIcon when unknown.
 * @type {Record<string, object>}
 */
export const nodeTypeIconMap = {
  server: Server,
  database: Library,
  schema: Layers,
  table: Grid,
  column: CodeSlash,
  action: AddCircle,
}

/** Used when a plugin node has no recognised node_type value. */
export const nodeTypeFallbackIcon = Document

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
