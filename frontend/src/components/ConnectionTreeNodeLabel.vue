<template>
  <div class="flex items-center justify-between w-full group/tree-node pr-1">
    <!-- node label -->
    <n-ellipsis class="flex-1 min-w-0 text-sm">{{ label }}</n-ellipsis>

    <!-- action buttons — revealed on hover via CSS group -->
    <div
      v-if="actions && actions.length > 0"
      class="flex items-center gap-0.5 opacity-0 group-hover/tree-node:opacity-100 transition-opacity flex-shrink-0 ml-1"
    >
      <n-tooltip
        v-for="action in actions"
        :key="action.type"
        :delay="600"
      >
        <template #trigger>
          <n-button
            size="tiny"
            :type="isDestructive(action) ? 'error' : 'default'"
            :secondary="!isDestructive(action)"
            @click.stop="emit('action', action)"
          >
            <template #icon>
              <n-icon>
                <component :is="iconFor(action)" />
              </n-icon>
            </template>
          </n-button>
        </template>
        {{ action.title || action.type }}
      </n-tooltip>
    </div>
  </div>
</template>

<script setup>
import { actionTypeIconMap, actionTypeFallbackIcon } from "@/lib/icons"

defineProps({
  /** Display label for the tree node */
  label: {
    type: String,
    required: true,
  },
  /**
   * Array of ConnectionTreeAction objects ({ type, title, query }) as returned
   * by the plugin's ConnectionTree response.
   */
  actions: {
    type: Array,
    default: () => [],
  },
})

const emit = defineEmits(["action"])

const DESTRUCTIVE_TYPES = new Set(["drop-database", "drop-table"])

function isDestructive(action) {
  return DESTRUCTIVE_TYPES.has(action.type)
}

function iconFor(action) {
  return actionTypeIconMap[action.type] ?? actionTypeFallbackIcon
}
</script>
