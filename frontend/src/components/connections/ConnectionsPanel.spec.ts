import { describe, expect, it } from 'vitest'
import { createPluginMap } from './ConnectionsPanel.vue'

// ensure pluginMap helper lowercases ids and preserves objects

describe('connectionsPanel helper', () => {
  it('createPluginMap lowercases plugin IDs', () => {
    const plugins = [
      { id: 'MySQL', foo: 'bar' },
      { id: 'postgresql', baz: 123 },
    ]
    const m = createPluginMap(plugins)
    expect(Object.keys(m)).toEqual(['mysql', 'postgresql'])
    expect(m.mysql.foo).toBe('bar')
    expect(m.postgresql.baz).toBe(123)
  })

  it('empty or missing list returns empty map', () => {
    expect(createPluginMap([])).toEqual({})
    expect(createPluginMap(null as any)).toEqual({})
    expect(createPluginMap(undefined as any)).toEqual({})
  })
})
