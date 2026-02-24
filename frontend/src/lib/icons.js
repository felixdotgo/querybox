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
  // tree / navigation
  LayersOutline,
  ServerOutline,
  LibraryOutline,
  GridOutline,
  CodeSlashOutline,
  DocumentOutline,
  ChevronDownOutline,
  ArrowDownOutline,

  // actions
  AddCircleOutline,
  EyeOutline,
  FlashOutline,
  RefreshOutline,
  TrashOutline,
} from "@vicons/ionicons5"

export {
  LayersOutline,     // driver group node
  ServerOutline,     // connection node
  LibraryOutline,    // node_type === "database"
  GridOutline,       // node_type === "table"
  CodeSlashOutline,  // node_type === "column"
  DocumentOutline,   // unknown / generic fallback
  ChevronDownOutline,// footer collapse toggle (rotate -90deg when collapsed)
  ArrowDownOutline,  // log panel auto-scroll toggle

  AddCircleOutline,  // new connection toolbar button
  EyeOutline,        // "select" action on tree nodes
  FlashOutline,      // "Connect" action on connection row
  RefreshOutline,    // "Refresh" action on connection row
  TrashOutline,      // "Delete" action on connection row
}

/**
 * Maps node_type strings (as returned by plugins via the proto node_type field)
 * to their icon component.  Falls back to nodeTypeFallbackIcon when unknown.
 * @type {Record<string, object>}
 */
export const nodeTypeIconMap = {
  server:   ServerOutline,
  database: LibraryOutline,
  schema:   LayersOutline,
  table:    GridOutline,
  column:   CodeSlashOutline,
}

/** Used when a plugin node has no recognised node_type value. */
export const nodeTypeFallbackIcon = DocumentOutline

/**
 * Maps ConnectionTreeAction type strings (constants defined in pkg/plugin/plugin.go)
 * to their icon component.  Falls back to actionTypeFallbackIcon when unknown.
 * @type {Record<string, object>}
 */
export const actionTypeIconMap = {
  "select":          EyeOutline,
  "describe":        CodeSlashOutline,
  "create-database": AddCircleOutline,
  "create-table":    AddCircleOutline,
  "drop-database":   TrashOutline,
  "drop-table":      TrashOutline,
}

/** Used when an action has no recognised type value. */
export const actionTypeFallbackIcon = FlashOutline
