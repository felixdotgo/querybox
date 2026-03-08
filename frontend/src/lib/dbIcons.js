// dbIcons.js - database-specific icons used in the connections list/tree
//
// Icons are pulled from the `simple-icons` package because it provides
// monochrome SVGs under an MIT license.  The SVGs do *not* hardcode a fill
// colour, which means consumers can recolour them via `currentColor` or by
// applying a `color` style to their container.
//
// To add a new driver icon:
//   1. Install or update the `simple-icons` package (`npm install simple-icons`).
//   2. Add the appropriate import above and include it in `driverIconMap`, keyed
//      by the string that appears in the connection's `driver_type` field.
//   3. If the simple-icons name differs from your driver string you can alias it.
//   4. Optionally update docs (see docs/features/01-connection-management.md).
//
// The frontend component `DbIcon.vue` consumes this map and automatically
// falls back to the generic `Server` icon if the driver is unknown.

import {
  siArangodb,
  siMongodb,
  siMysql,
  siPostgresql,
  siRedis,
  siSqlite,
} from 'simple-icons'

// This list should mirror the set of drivers for which we expect icons.
// Tests in `dbIcons.test.ts` make sure the map is not accidentally empty.
export const driverIconMap = {
  postgresql: siPostgresql,
  mysql: siMysql,
  mongodb: siMongodb,
  sqlite: siSqlite,
  redis: siRedis,
  arangodb: siArangodb,
}

// helper to return a simple-icons entry or undefined
export function getDriverIcon(driverType) {
  if (!driverType || typeof driverType !== 'string')
    return undefined
  return driverIconMap[driverType.toLowerCase()]
}

/**
 * Determine the effective icon name for a given driver type, taking an
 * optional plugin object (as returned by usePlugins) into account.
 *
 * @param {string} driverType - value stored in connection.driver_type
 * @param {object} [plugin] - plugin metadata object; may be undefined
 * @returns {string} icon key suitable for passing to `DbIcon`/`getDriverIcon`
 */
export function getIconNameForDriver(driverType, plugin) {
  if (plugin && plugin.metadata && typeof plugin.metadata.simple_icon === 'string' && plugin.metadata.simple_icon.trim() !== '') {
    return plugin.metadata.simple_icon.toLowerCase()
  }
  return driverType ? driverType.toLowerCase() : ''
}
