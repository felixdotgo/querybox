<script setup>
import { NIcon } from 'naive-ui'
import { computed, h } from 'vue'
import { actionTypeFallbackIcon, actionTypeIconMap, EllipsisHorizontal } from '@/lib/icons'

const props = defineProps({
  /** Display label for the tree node */
  label: {
    type: String,
    required: true,
  },
  /**
   * Array of ConnectionTreeAction objects ({ type, title, query, hidden }) as
   * returned by the plugin's ConnectionTree response.
   */
  actions: {
    type: Array,
    default: () => [],
  },
})

const emit = defineEmits(['action'])

/** Actions that should appear in the dropdown (hidden ones fire on click). */
const visibleActions = computed(() => props.actions.filter(a => !a.hidden))

const DESTRUCTIVE_TYPES = new Set(['drop-database', 'drop-table', 'drop-collection'])

function renderIcon(icon) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

const menuOptions = computed(() => {
  const items = []
  visibleActions.value.forEach((action, i) => {
    if (i > 0 && DESTRUCTIVE_TYPES.has(action.type) && !DESTRUCTIVE_TYPES.has(visibleActions.value[i - 1].type)) {
      items.push({ type: 'divider', key: `divider-${i}` })
    }
    items.push({
      key: i,
      label: action.title || action.type,
      icon: renderIcon(actionTypeIconMap[action.type] ?? actionTypeFallbackIcon),
    })
  })
  return items
})

function handleMenuSelect(key) {
  emit('action', visibleActions.value[key])
}
</script>

<template>
  <div class="flex items-center justify-between w-full group/tree-node pr-1">
    <!-- node label -->
    <n-ellipsis class="flex-1 min-w-0 text-sm">
      {{ label }}
    </n-ellipsis>

    <!-- three-dot context menu — revealed on hover via CSS group.
         Hidden actions (hidden: true) are excluded; they fire on node click. -->
    <div
      v-if="visibleActions.length > 0"
      class="opacity-0 group-hover/tree-node:opacity-100 transition-opacity flex-shrink-0 ml-1"
    >
      <n-dropdown
        trigger="click"
        :options="menuOptions"
        placement="bottom-end"
        @select="handleMenuSelect"
      >
        <n-button
          size="tiny"
          quaternary
          @click.stop
        >
          <template #icon>
            <NIcon><EllipsisHorizontal /></NIcon>
          </template>
        </n-button>
      </n-dropdown>
    </div>
  </div>
</template>
