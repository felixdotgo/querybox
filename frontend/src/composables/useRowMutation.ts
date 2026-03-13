import { GetCredential } from '@/bindings/github.com/felixdotgo/querybox/services/connectionservice'
import { MutateRow } from '@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager'

/**
 * Helper for performing a row mutation via the plugin manager.
 *
 * The `conn` object should be the same connection record used elsewhere in
 * the application (it must include `id` and `driver_type`).  `operation`
 * should be the numeric enum value defined by the backend (e.g.
 * `1` for INSERT, `2` for UPDATE, `3` for DELETE).  The remaining fields
 * are forwarded verbatim to the plugin.  A prepared credential blob will be
 * injected automatically.
 *
 * Returns the plugin's MutateRowResponse promise.
 */
export async function mutateRow(conn: { id: string, driver_type: string }, operation: number, source: string, values: Record<string, string> = {}, filter: string = '') {
  if (!conn || !conn.id || !conn.driver_type) {
    throw new Error('invalid connection for mutateRow')
  }

  const params: Record<string, any> = {}
  const cred = await GetCredential(conn.id)
  if (cred) {
    params.credential_blob = cred
  }

  // When the source is a qualified name (e.g. "employees.users"), extract the
  // database prefix and forward it so the plugin can select the correct
  // database even when no default database is saved in the credential.
  if (source && source.includes('.')) {
    const dbName = source.split('.')[0]
    if (dbName)
      params.database = dbName
  }

  // the binding expects the connection map followed by the other parameters
  return await MutateRow(conn.driver_type, params, operation, source, values, filter)
}
