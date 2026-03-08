import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import WorkspacePanel from './WorkspacePanel.vue'

// we will mock the composable so tests can control currentSchema behavior
vi.mock('@/composables/useConnectionTree', () => ({
  useConnectionTree: (connRef: any) => ({
    getSchema: vi.fn(),
    fetchSchema: vi.fn(),
  }),
}))

// helper to override the mock implementation on a per-test basis
function setSchemaHelpers(getSchema: any, fetchSchema: any) {
  const mod = require('@/composables/useConnectionTree')
  mod.useConnectionTree = () => ({ getSchema, fetchSchema })
}

describe('workspacePanel.vue', () => {
  it('defaults new tabs to result view even when schema is cached', async () => {
    const getSchema = vi.fn().mockReturnValue({ name: 'foo' })
    const fetchSchema = vi.fn()
    setSchemaHelpers(getSchema, fetchSchema)

    const wrapper: any = mount(WorkspacePanel, {
      props: { selectedConnection: null },
    })

    // open a tab pointing at a table for which getSchema returns data
    wrapper.vm.openTab(
      'foo',
      { rows: [] },
      null,
      'key',
      Date.now(),
      {
        conn: { id: 'c1', driver_type: 'dummy' },
        node: { key: 'c1:foo' },
      },
    )
    await wrapper.vm.$nextTick()

    const tab = wrapper.vm.tabs.find((t: any) => t.key === 'key')
    expect(tab).toBeTruthy()
    expect(tab.innerTab).toBe('result')
  })

  it('never renders a "Structure" toolbar button even when schema available', async () => {
    const getSchema = vi.fn().mockReturnValue({ name: 'foo' })
    const fetchSchema = vi.fn()
    setSchemaHelpers(getSchema, fetchSchema)

    const wrapper = mount(WorkspacePanel, {
      props: { selectedConnection: null },
    })

    wrapper.vm.openTab(
      'foo',
      { rows: [] },
      null,
      'key',
      Date.now(),
      {
        conn: { id: 'c1', driver_type: 'dummy' },
        node: { key: 'c1:foo' },
      },
    )
    await wrapper.vm.$nextTick()

    // the toolbar text should not contain the word "Structure"
    expect(wrapper.text()).not.toContain('Structure')
  })

  it('passes the correct connection object to schema helpers when switching tabs', async () => {
    const getSchema = vi.fn()
    const fetchSchema = vi.fn()
    setSchemaHelpers(getSchema, fetchSchema)

    const wrapper: any = mount(WorkspacePanel, {
      props: { selectedConnection: null },
    })

    const connA = { id: 'A', driver_type: 'mysql' }
    const connB = { id: 'B', driver_type: 'mongodb' }

    // open a tab for connection A; prefetch logic should call fetchSchema with that conn
    wrapper.vm.openTab('a', { rows: [] }, null, 'kA', Date.now(), {
      conn: connA,
      node: { key: 'A:table1' },
    })
    await wrapper.vm.$nextTick()

    expect(fetchSchema).toHaveBeenCalledWith('table1', connA)
    // also getSchema may have been consulted, ensure it used connA
    expect(getSchema).toHaveBeenCalledWith('table1', connA)

    fetchSchema.mockClear()
    getSchema.mockClear()

    // open a second tab for connection B, this should trigger fetchSchema with connB
    wrapper.vm.openTab('b', { rows: [] }, null, 'kB', Date.now(), {
      conn: connB,
      node: { key: 'B:coll' },
    })
    await wrapper.vm.$nextTick()

    expect(fetchSchema).toHaveBeenCalledWith('coll', connB)
    expect(getSchema).toHaveBeenCalledWith('coll', connB)

    fetchSchema.mockClear()
    getSchema.mockClear()

    // switch back to the first tab by updating activeTabKey manually;
    // the watcher should again pass connA when deciding to fetch
    wrapper.vm.activeTabKey = 'kA'
    await wrapper.vm.$nextTick()
    expect(fetchSchema).toHaveBeenCalledWith('table1', connA)
  })
})
