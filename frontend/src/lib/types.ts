/** Persisted connection record returned by the backend. */
export interface Connection {
  id: string
  name: string
  driver_type: string
  credential_key: string
  created_at: string
  updated_at: string
}

/** Plugin metadata discovered by the backend plugin manager. */
export interface PluginInfo {
  id: string
  name: string
  path: string
  running: boolean
  type?: number
  version?: string
  description?: string
  url?: string
  author?: string
  capabilities?: string[]
  tags?: string[]
  license?: string
  icon_url?: string
  contact?: string
  metadata?: Record<string, string>
  settings?: Record<string, string>
  lastError?: string
}

/** A single action exposed by a connection tree node. */
export interface TreeAction {
  type: string
  title?: string
  query?: string
  new_tab?: boolean
  fields?: TreeActionField[]
}

/** An input field definition for tree actions that require user input. */
export interface TreeActionField {
  name: string
  label?: string
  placeholder?: string
  required?: boolean
}

/** A node in the hierarchical connection tree returned by plugins. */
export interface TreeNode {
  key: string
  label: string
  node_type: string | number
  children?: TreeNode[]
  actions?: TreeAction[]
  /** Injected by tagWithConnId — the owning connection ID. */
  _connectionId?: string
}

/** Execution context attached to a workspace tab. */
export interface TabContext {
  conn: Connection
  action: TreeAction
  node: TreeNode
  capabilities: string[]
  explain?: boolean
}

/** Parameters for opening a workspace tab. */
export interface OpenTabParams {
  title: string
  result?: ExecResult | null
  error?: string | null
  tabKey?: string
  version?: number
  context?: TabContext
}

/** A workspace tab. */
export interface Tab {
  key: string
  title: string
  type?: string
  result: ExecResult | null
  error: string | null
  explainResult?: ExecResult | null
  explainError?: string | null
  innerTab: string
  version: number
  context?: TabContext
  loading: boolean
  query: string
  language: string
}

/** SQL result payload from a plugin. */
export interface SqlResult {
  columns?: Column[]
  rows?: Row[]
}

/** A column descriptor in a SQL result. */
export interface Column {
  name: string
  type?: string
}

/** A row in a SQL result. */
export interface Row {
  values?: string[]
}

/** Document result payload (JSON documents). */
export interface DocumentResult {
  documents?: string[]
}

/** Key-value result payload. */
export interface KeyValueResult {
  data?: Record<string, string>
}

/** Union of possible result payloads from a plugin exec response. */
export type ExecResult = SqlResult | DocumentResult | KeyValueResult

/** Table schema metadata from DescribeSchema. */
export interface TableSchema {
  name: string
  columns?: ColumnSchema[]
  indexes?: IndexSchema[]
}

/** Column schema metadata. */
export interface ColumnSchema {
  name: string
  type: string
  nullable: boolean
  primary_key: boolean
}

/** Index schema metadata. */
export interface IndexSchema {
  name: string
  columns: string[]
  unique: boolean
}

/** Structured log entry emitted by the backend. */
export interface LogEntry {
  level: 'debug' | 'info' | 'warn' | 'error'
  message: string
  timestamp: string
}

/** Auth field type enum — mirrors PluginV1_AuthField_FieldType. */
export enum AuthFieldType {
  UNKNOWN = 0,
  TEXT = 1,
  NUMBER = 2,
  PASSWORD = 3,
  CHECKBOX = 4,
  SELECT = 5,
  FILE_PATH = 6,
}

/** A single input field for a plugin auth form. */
export interface AuthField {
  type?: AuthFieldType
  name?: string
  label?: string
  value?: string
  required?: boolean
  options?: (AuthFieldOption | null)[]
}

/** An option for a SELECT auth field. */
export interface AuthFieldOption {
  label?: string
  value?: string
}

/** A set of fields for one authentication method (e.g. "basic", "oauth"). */
export interface AuthForm {
  key?: string
  name?: string
  fields?: (AuthField | null)[]
}

/** Saved credential blob serialized by useAuthForms. */
export interface SavedCredential {
  form?: string
  values?: Record<string, string>
}

/** Parameters passed to the row editor mutation handler. */
export interface MutationParams {
  operation: string
  source: string
  values?: Record<string, string>
  filter: string
}

/** Backend event names — mirrors services/events.go constants. */
export const EventNames = {
  AppLog: 'app:log',
  ConnectionCreated: 'connection:created',
  ConnectionUpdated: 'connection:updated',
  ConnectionDeleted: 'connection:deleted',
  MenuLogsToggled: 'menu:logs-toggled',
  ConnectionsWindowClosed: 'connections-window:closed',
  EditConnectionWindowOpened: 'edit-connection-window:opened',
  EditConnectionWindowClosed: 'edit-connection-window:closed',
  PluginsReady: 'plugins:ready',
} as const
