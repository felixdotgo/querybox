<script setup>
import { NButton, NIcon } from 'naive-ui'
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import {
  ShowConnectionsWindow,
} from '@/bindings/github.com/felixdotgo/querybox/services/app'
import { tagWithConnId, useConnectionTree } from '@/composables/useConnectionTree'
import { useConnectionEvents } from '@/composables/useConnectionEvents'
import { useTreeRenderers } from '@/composables/useTreeRenderers'
import { useTreeActions } from '@/composables/useTreeActions'
import { usePlugins } from '@/composables/usePlugins'
import { AddCircle, Search } from '@/lib/icons'
import { ShowEditConnectionWindow } from '@/bindings/github.com/felixdotgo/querybox/services/app'
import ActionFormModal from './ActionFormModal.vue'

const props = defineProps({
  activeConnectionId: { type: String, default: null },
})

const emit = defineEmits([
  'connection-selected',
  'query-result',
  'connection-opened',
])

const router = useRouter()
async function openConnections() {
  try {
    await ShowConnectionsWindow()
  }
  catch {
    router.push('/connections')
  }
}

// panel state -------------------------------------------------------------
const treeScrollRef = ref(null)
const isScrolled = ref(false)
const connections = ref([])
const { plugins } = usePlugins()
const pluginCaps = computed(() => {
  const map = {}
  for (const p of plugins.value || []) {
    if (p && p.id) {
      map[p.id] = p.capabilities || []
    }
  }
  return map
})

const pluginMap = computed(() => {
  const m = {}
  for (const p of plugins.value || []) {
    if (p && p.id) {
      m[p.id.toLowerCase()] = p
    }
  }
  return m
})

const loadingNodes = ref({})
const connecting = ref({})
const filter = ref('')
const { cache: connectionTrees, load: loadConnectionTree, schemaCache } = useConnectionTree()
const selectedConnection = ref(null)
const expandedKeys = ref([])

// --- Event handling (extracted to composable) ---
const { loadConnections } = useConnectionEvents({
  connections,
  connectionTrees,
  schemaCache,
  selectedConnection,
  expandedKeys,
  loadingNodes,
  filter,
})

// --- Tree actions (extracted to composable) ---
const {
  deleteModal,
  actionModal,
  runTreeAction,
  fetchTreeFor,
  handleAction,
  handleSelect,
  handleConnectionDblclick,
  onActionModalSubmit,
  confirmDelete,
} = useTreeActions({
  connections,
  connectionTrees,
  schemaCache,
  expandedKeys,
  loadingNodes,
  connecting,
  selectedConnection,
  pluginCaps,
  loadConnectionTree,
  emit,
})

const defaultExpandedKeys = computed(() => {
  const ids = new Set(Object.keys(connectionTrees))
  connections.value.forEach(c => ids.add(c.id))
  return Array.from(ids)
})

const treeData = computed(() => {
  return (connections.value || []).map((cc) => {
    const extra = tagWithConnId(connectionTrees[cc.id] || [], cc.id)
    return { key: cc.id, label: cc.name, children: extra.length ? extra : undefined }
  })
})

const filteredTreeData = computed(() => {
  const q = (filter.value || '').toLowerCase().trim()
  if (!q)
    return treeData.value
  return treeData.value.filter(node =>
    (node.label || '').toLowerCase().includes(q),
  )
})

// --- Tree renderers (extracted to composable) ---
const activeConnectionIdComputed = computed(() => props.activeConnectionId)

const { getNodeProps, renderLabel, renderPrefix } = useTreeRenderers({
  connections,
  connectionTrees,
  schemaCache,
  selectedConnection,
  loadingNodes,
  connecting,
  pluginMap,
  activeConnectionId: activeConnectionIdComputed,
  onConnect(conn) {
    if (connectionTrees[conn.id]) {
      delete connectionTrees[conn.id]
      delete schemaCache[conn.id]
    }
    fetchTreeFor(conn)
  },
  onEdit(conn) {
    ShowEditConnectionWindow(conn.id)
  },
  onDelete(conn) {
    deleteModal.value = { visible: true, conn }
  },
  onDblclick(conn) {
    handleConnectionDblclick(conn)
  },
  onAction(conn, action, node) {
    handleAction(conn, action, node)
  },
})

watch(filter, (q) => {
  if (!(q || '').trim() && treeScrollRef.value) {
    treeScrollRef.value.scrollTop = 0
  }
})

// initialize
loadConnections()

defineExpose({
  runTreeAction,
})
</script>

<template>
  <div class="p-3 h-full flex flex-col gap-3">
    <!-- small toolbar -->
    <div class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <span class="text-sm font-semibold m-0">Connections</span>
      </div>
      <div class="flex">
        <NButton
          size="tiny"
          type="primary"
          title="New connection"
          @click="openConnections"
        >
          <template #icon>
            <NIcon><AddCircle /></NIcon>
          </template>
        </NButton>
      </div>
    </div>

    <n-input
      v-model:value="filter"
      size="small"
      placeholder="Search connections..."
    >
      <template #prefix>
        <NIcon><Search /></NIcon>
      </template>
    </n-input>

    <div
      ref="treeScrollRef"
      class="flex-1 overflow-y-auto mt-2 px-1 min-h-0 transition-shadow duration-150 scroll-container"
      :class="{ 'shadow-[inset_0_6px_6px_-6px_rgba(0,0,0,0.12)]': isScrolled }"
      @scroll.passive="isScrolled = $event.target.scrollTop > 0"
    >
      <n-tree
        v-model:expanded-keys="expandedKeys"
        show-line
        :data="filteredTreeData"
        :default-expanded-keys="defaultExpandedKeys"
        :node-key="node => node.key"
        block-node
        :show-selector="false"
        :node-props="getNodeProps"
        :render-label="renderLabel"
        :render-prefix="renderPrefix"
        :indent="12"
        @update:selected-keys="handleSelect"
      />
      <div
        v-if="connections.length === 0"
        class="py-6 text-center opacity-70"
      >
        No connections yet
      </div>
    </div>

    <!-- action input form (create-database, create-table, …) -->
    <ActionFormModal
      v-model:visible="actionModal.visible"
      :action="actionModal.action"
      @submit="onActionModalSubmit"
    />

    <!-- delete confirmation dialog -->
    <n-modal
      v-model:show="deleteModal.visible"
      preset="dialog"
      type="error"
      title="Remove connection"
      :content="`Remove &quot;${deleteModal.conn?.name}&quot;? This cannot be undone.`"
      positive-text="Remove"
      negative-text="Cancel"
      @positive-click="confirmDelete"
      @negative-click="deleteModal.visible = false"
    />
  </div>
</template>

<style scoped>
.scroll-container::-webkit-scrollbar {
  width: 4px;
}
.scroll-container::-webkit-scrollbar-track {
  background: transparent;
}
.scroll-container::-webkit-scrollbar-thumb {
  background-color: transparent;
  border-radius: 9999px;
}
.scroll-container:hover::-webkit-scrollbar-thumb {
  background-color: rgba(0, 0, 0, 0.15);
}
</style>
