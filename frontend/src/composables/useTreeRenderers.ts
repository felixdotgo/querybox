import { NButton, NIcon, NSpin } from 'naive-ui'
import { h, type ComputedRef, type Ref } from 'vue'
import DbIcon from '@/components/DbIcon.vue'
import ConnectionEntryLabel from '@/components/connections/ConnectionEntryLabel.vue'
import ConnectionTreeItemLabel from '@/components/connections/ConnectionTreeItemLabel.vue'
import { getIconNameForDriver } from '@/lib/dbIcons'
import { nodeTypeFallbackIcon, nodeTypeIconMap } from '@/lib/icons'
import type { Connection, PluginInfo } from '@/lib/types'

interface TreeRenderOptions {
  connections: Ref<Connection[]>
  connectionTrees: Record<string, unknown>
  schemaCache: Record<string, unknown>
  selectedConnection: Ref<Connection | null>
  loadingNodes: Ref<Record<string, boolean>>
  connecting: Ref<Record<string, boolean>>
  pluginMap: ComputedRef<Record<string, PluginInfo>>
  activeConnectionId: ComputedRef<string | null>
  onConnect: (conn: Connection) => void
  onEdit: (conn: Connection) => void
  onDelete: (conn: Connection) => void
  onDblclick: (conn: Connection) => void
  onAction: (conn: Connection, action: any, node: any) => void
}

/**
 * Provides tree render functions (renderLabel, renderPrefix, getNodeProps)
 * for the Naive UI NTree in ConnectionsPanel.
 */
export function useTreeRenderers(opts: TreeRenderOptions) {
  const {
    connections,
    connectionTrees,
    schemaCache,
    loadingNodes,
    connecting,
    pluginMap,
    activeConnectionId,
    selectedConnection,
    onConnect,
    onEdit,
    onDelete,
    onDblclick,
    onAction,
  } = opts

  function getNodeProps(node: any) {
    const props: Record<string, any> = {}
    const conn = connections.value.find((c: Connection) => c.id === node.key)
    if (conn) {
      props.onDblclick = (e: Event) => {
        e.stopPropagation()
        onDblclick(conn)
      }
    }
    return props
  }

  function renderLabel({ option }: { option: any }) {
    const conn = connections.value.find((c: Connection) => c.id === option.key)

    // "action" leaf nodes
    if (!conn && option.node_type === 'action') {
      const actionIcon = (nodeTypeIconMap as Record<string, any>)[option.node_type] ?? nodeTypeFallbackIcon
      return h(
        NButton,
        { size: 'tiny', secondary: true, style: 'margin: 1px 0', type: 'primary' },
        {
          icon: () => h(NIcon, { size: 14 }, { default: () => h(actionIcon) }),
          default: () => option.label,
        },
      )
    }

    // non-connection nodes with actions
    if (!conn && option.actions && option.actions.length > 0) {
      return h(ConnectionTreeItemLabel, {
        label: option.label,
        actions: option.actions,
        onAction(action: any) {
          const parentConn = option._connectionId
            ? connections.value.find((c: Connection) => c.id === option._connectionId)
            : selectedConnection.value
          if (parentConn)
            onAction(parentConn, action, option)
        },
      })
    }

    if (!conn)
      return option.label

    return h(ConnectionEntryLabel, {
      label: option.label,
      hasTree: !!connectionTrees[conn.id],
      isActive: activeConnectionId.value === conn.id,
      loading: !!connecting.value[conn.id],
      onConnect() {
        if (connectionTrees[conn.id]) {
          delete connectionTrees[conn.id]
          delete schemaCache[conn.id]
        }
        onConnect(conn)
      },
      onEdit() {
        onEdit(conn)
      },
      onDelete() {
        onDelete(conn)
      },
      onDblclick() {
        onDblclick(conn)
      },
    })
  }

  function renderPrefix({ option }: { option: any }) {
    if (loadingNodes.value[option.key]) {
      return h(NSpin, { size: 14 })
    }

    if (option.node_type === 'action')
      return null

    const conn = connections.value.find((c: Connection) => c.id === option.key)
    if (conn) {
      const key = conn.driver_type ? conn.driver_type.toLowerCase() : ''
      const plugin = pluginMap.value[key]
      const iconName = getIconNameForDriver(conn.driver_type, plugin)
      return h(DbIcon, { driver: iconName, size: 14 })
    }

    const icon = (nodeTypeIconMap as Record<string, any>)[option.node_type] ?? nodeTypeFallbackIcon
    const iconNode = h(NIcon, { size: 14 }, { default: () => h(icon) })

    if (conn && activeConnectionId.value === (conn as Connection).id) {
      return h('div', { style: { position: 'relative', display: 'inline-flex' } }, [
        iconNode,
        h('span', {
          style: {
            position: 'absolute',
            bottom: '-2px',
            right: '-3px',
            width: '8px',
            height: '8px',
            borderRadius: '50%',
            backgroundColor: '#22c55e',
            border: '1px solid white',
          },
        }),
      ])
    }

    return iconNode
  }

  return { getNodeProps, renderLabel, renderPrefix }
}
