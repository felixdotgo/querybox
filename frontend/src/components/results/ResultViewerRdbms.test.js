import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { getDataTypeColor } from '@/lib/icons'
import ResultViewerRdbms from './ResultViewerRdbms.vue'

// utility to inspect rendered tags or icons by class
function findDatatypeBadge(wrapper) {
  return wrapper.find('.datatype-badge')
}
function findPrimaryKeyIcon(wrapper) {
  return wrapper.find('.primary-key-icon')
}

describe('resultViewerRdbms', () => {
  it('renders a colored badge when schema.type is provided', () => {
    const payload = {
      columns: [{ name: 'age' }],
      rows: [[{ value: 42 }]],
    }
    const schema = {
      columns: [{ name: 'age', type: 'integer' }],
    }

    const wrapper = mount(ResultViewerRdbms, {
      props: { payload, schema },
      global: {
        stubs: { 'n-data-table': true },
        components: { 'n-icon': true, 'n-tag': true },
      },
    })

    const badge = findDatatypeBadge(wrapper)
    expect(badge.exists()).toBe(true)
    expect(badge.classes()).toContain('datatype-badge')
    expect(badge.text()).toBe('integer')
    const expectedColor = getDataTypeColor('integer')
    expect(badge.attributes('style')).toContain(expectedColor)

    // verify length/precision stripping
    const wrapper2 = mount(ResultViewerRdbms, {
      props: {
        payload: { columns: [{ name: 'name' }], rows: [[{ value: 'x' }]] },
        schema: { columns: [{ name: 'name', type: 'varchar(14)' }] },
      },
      global: {
        stubs: { 'n-data-table': true },
        components: { 'n-icon': true, 'n-tag': true },
      },
    })
    const badge2 = findDatatypeBadge(wrapper2)
    expect(badge2.exists()).toBe(true)
    expect(badge2.text()).toBe('varchar')
    expect(badge2.attributes('style')).toContain(getDataTypeColor('varchar'))
  })

  it('shows a primary-key icon when primary_key is true', () => {
    const payload = {
      columns: [{ name: 'id' }],
      rows: [[{ value: 1 }]],
    }
    const schema = {
      columns: [{ name: 'id', primary_key: true }],
    }

    const wrapper = mount(ResultViewerRdbms, {
      props: { payload, schema },
      global: {
        stubs: { 'n-data-table': true },
        components: { 'n-icon': true, 'n-tag': true },
      },
    })

    expect(findPrimaryKeyIcon(wrapper).exists()).toBe(true)
  })

  it('does not render a datatype badge when type is missing', () => {
    const payload = {
      columns: [{ name: 'foo' }],
      rows: [[{ value: 'bar' }]],
    }
    const schema = { columns: [{ name: 'foo' }] }

    const wrapper = mount(ResultViewerRdbms, {
      props: { payload, schema },
      global: { stubs: { 'n-data-table': true }, components: { 'n-icon': true, 'n-tag': true } },
    })

    expect(findDatatypeBadge(wrapper).exists()).toBe(false)
  })
})
