import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'

import { DescribeSchema } from '@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager'
import { tagWithConnId, useConnectionTree } from './useConnectionTree'

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

    // override connection should use different id and therefore return null
    const override = { id: 'other', driver_type: 'dummy' }
    expect(getSchema('public.users', override)).toBeNull()

    // if we populate cache for the override ID we should get results
    schemaCache.other = { 'public.users': { name: 'public.users', cols: [] } }
    expect(getSchema('public.users', override)).toEqual({ name: 'public.users', cols: [] })
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

  it('tagWithConnId correctly tags nodes and removes duplicates', () => {
    // prepare a small tree containing two siblings with the same key; the
    // dedup logic should keep only the first, prefix every key with the
    // connection id, and recurse into children.
    const sample = [
      { key: 'pub', label: 'public', node_type: 4, children: [
        { key: 'users', label: 'users', node_type: 2 },
      ] },
      { key: 'pub', label: 'should be dropped', node_type: 4 },
    ]
    const result = tagWithConnId(sample, 'conn1')
    expect(result.length).toBe(1)
    expect(result[0].key).toBe('conn1:pub')
    expect(result[0].node_type).toBe('schema')
    expect(result[0].children && result[0].children[0].key).toBe('conn1:pub:users')

    // invoking again on the same input should produce the same output (no
    // cumulative duplicates)
    const again = tagWithConnId(sample, 'conn1')
    expect(again).toEqual(result)
  })

  it('tagWithConnId gives unique keys to same-named schemas in different databases', () => {
    // Two database nodes each containing a "public" schema child.  After tagging,
    // the schema keys must differ so that NaiveUI tracks their expansion state
    // independently (the original bug: both resolved to "connId:public").
    const tree = [
      {
        key: 'nguye',
        label: 'nguye',
        node_type: 1,
        children: [
          { key: 'public', label: 'public', node_type: 4, children: [
            { key: 'public.users', label: 'users', node_type: 2 },
          ] },
        ],
      },
      {
        key: 'phonedb',
        label: 'phonedb',
        node_type: 1,
        children: [
          { key: 'public', label: 'public', node_type: 4, children: [
            { key: 'public.users', label: 'users', node_type: 2 },
          ] },
        ],
      },
    ]
    const result = tagWithConnId(tree, 'c1')

    const nguyeSchema = result.find(n => n.label === 'nguye')!.children[0]
    const phonedbSchema = result.find(n => n.label === 'phonedb')!.children[0]

    // database-level keys must be unique
    expect(result[0].key).toBe('c1:nguye')
    expect(result[1].key).toBe('c1:phonedb')

    // schema keys must be unique across databases
    expect(nguyeSchema.key).toBe('c1:nguye:public')
    expect(phonedbSchema.key).toBe('c1:phonedb:public')
    expect(nguyeSchema.key).not.toBe(phonedbSchema.key)

    // table keys must also be unique
    expect(nguyeSchema.children[0].key).toBe('c1:nguye:public:public.users')
    expect(phonedbSchema.children[0].key).toBe('c1:phonedb:public:public.users')
    expect(nguyeSchema.children[0].key).not.toBe(phonedbSchema.children[0].key)

    // each node must carry the correct connection id regardless of depth
    expect(nguyeSchema._connectionId).toBe('c1')
    expect(phonedbSchema._connectionId).toBe('c1')
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

    // override the connection and ensure its driver_type is used
    spy.mockClear()
    const override = { id: 'c3', driver_type: 'bananas' }
    await fetchSchema('foo.bar', override)
    expect(spy).toHaveBeenCalledWith('bananas', expect.any(Object), 'foo', 'bar')
  })
})
