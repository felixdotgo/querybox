import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'

import { DescribeSchema } from '@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager'
import { useConnectionTree } from './useConnectionTree'

// stub out bindings that pull in the Wails runtime, which assumes a browser
// `window` object; this prevents a ReferenceError in the Node test
// environment. only the signatures used by the composable are needed.
vi.mock('@/bindings/github.com/felixdotgo/querybox/services/connectionservice', () => ({
  GetCredential: vi.fn(),
}))
vi.mock('@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager', () => ({
  DescribeSchema: vi.fn(),
  GetConnectionTree: vi.fn(),
}))

// These tests focus solely on the client-side caching/lookup behavior that
// triggered the "missing Structure tab" bug.  We avoid invoking any network
// helpers by manipulating the reactive cache directly.

describe('useConnectionTree schema helpers', () => {
  it('getSchema resolves both full and suffix keys', () => {
    const connRef = ref({ id: 'c1' })
    const { getSchema, schemaCache } = useConnectionTree(connRef)

    // populate with a few representative entries
    schemaCache.c1 = {
      'public.users': { name: 'public.users', cols: [] },
      'users': { name: 'public.users', cols: [] },
      'foo.bar.baz': { name: 'foo.bar.baz', cols: [] },
    }

    expect(getSchema('public.users')).toEqual({ name: 'public.users', cols: [] })
    expect(getSchema('users')).toEqual({ name: 'public.users', cols: [] })
    // suffix-only lookup (first dot removed) should still return the object
    expect(getSchema('bar.baz')).toEqual({ name: 'foo.bar.baz', cols: [] })
    // final fallback walks keys for an "endsWith" match
    expect(getSchema('baz')).toEqual({ name: 'foo.bar.baz', cols: [] })
    // unknown table should return null
    expect(getSchema('nope')).toBeNull()
  })

  it('fetchSchema merges new entries without overwriting existing cache', async () => {
    const connRef = ref({ id: 'c2', driver_type: 'dummy' })
    const { getSchema, schemaCache, fetchSchema } = useConnectionTree(connRef)

    // prime cache with one table
    schemaCache.c2 = { initial: { name: 'initial' } }

    // stub DescribeSchema to return one new table
    const fake = { tables: [{ name: 'newtable' }] }
    ;(DescribeSchema as any).mockResolvedValue(fake)

    await fetchSchema('newtable')
    expect(schemaCache.c2.initial).toBeDefined()
    expect(schemaCache.c2.newtable).toBeDefined()

    // calling again should merge but not duplicate
    await fetchSchema('newtable')
    expect(Object.keys(schemaCache.c2)).toEqual(expect.arrayContaining(['initial', 'newtable']))
  })

  it('fetchSchema splits qualified name into db and table filters', async () => {
    const connRef = ref({ id: 'c3', driver_type: 'dummy' })
    const { fetchSchema } = useConnectionTree(connRef)

    // spy on DescribeSchema so we can inspect arguments
    const spy = vi.spyOn({ DescribeSchema }, 'DescribeSchema' as any)
    const fake = { tables: [] }
    ;(DescribeSchema as any).mockResolvedValue(fake)

    await fetchSchema('foo.bar')
    expect(spy).toHaveBeenCalledWith('dummy', expect.any(Object), 'foo', 'bar')
  })
})
