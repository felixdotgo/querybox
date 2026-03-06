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
})
