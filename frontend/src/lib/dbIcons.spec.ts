import { describe, expect, it } from 'vitest'
import { driverIconMap, getDriverIcon, getIconNameForDriver } from './dbIcons'

// Basic sanity tests for the icon helpers introduced in the connection UX

describe('dbIcons helpers', () => {
  it('driverIconMap contains at least one mapping', () => {
    expect(Object.keys(driverIconMap).length).toBeGreaterThan(0)
  })

  it('getDriverIcon returns an object for known keys and undefined for unknown', () => {
    // pick one key from the map
    const keys = Object.keys(driverIconMap)
    const sample = keys[0]
    expect(getDriverIcon(sample)).toBeDefined()
    expect(getDriverIcon(sample.toUpperCase())).toBeDefined() // case-insensitive

    expect(getDriverIcon('not-a-driver')).toBeUndefined()
    expect(getDriverIcon('')).toBeUndefined()
  })

  it('getIconNameForDriver uses plugin.metadata.simple_icon when present', () => {
    const base = 'mysql'
    const plugin = { metadata: { simple_icon: 'PostgreSQL' } }
    expect(getIconNameForDriver(base, plugin)).toBe('postgresql')

    // missing metadata falls back to driver type
    expect(getIconNameForDriver(base, {} as any)).toBe('mysql')
    expect(getIconNameForDriver(base, null as any)).toBe('mysql')

    // empty string driver returns empty string
    expect(getIconNameForDriver('', plugin)).toBe('')
  })

  it('returned icon name may not exist in the map', () => {
    const base = 'mysql'
    const plugin = { metadata: { simple_icon: 'nonexistent' } }
    const name = getIconNameForDriver(base, plugin)
    expect(name).toBe('nonexistent')
    expect(getDriverIcon(name)).toBeUndefined()
  })
})
