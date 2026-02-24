<template>
  <div
    class="flex items-center justify-between w-full group/conn pr-1"
    @dblclick.stop="emit('dblclick')"
  >
    <!-- connection name -->
    <n-ellipsis class="flex-1 min-w-0 text-sm">{{ label }}</n-ellipsis>

    <!-- action buttons — revealed on hover via CSS group -->
    <div
      class="flex items-center gap-0.5 opacity-0 group-hover/conn:opacity-100 transition-opacity flex-shrink-0 ml-1"
    >
      <!-- Connect / Refresh -->
      <n-tooltip :delay="600">
        <template #trigger>
          <n-button
            size="tiny"
            :type="hasTree ? 'default' : 'primary'"
            primary
            @click.stop="emit('connect')"
          >
            <template #icon>
              <n-icon><component :is="hasTree ? RefreshOutline : FlashOutline" /></n-icon>
            </template>
             {{ hasTree ? "Reconnect" : "Connect" }}
          </n-button>
        </template>
        {{ hasTree ? "Reconnect" : "Connect" }}
      </n-tooltip>
      &nbsp;
      <!-- Delete -->
      <n-tooltip :delay="600">
        <template #trigger>
          <n-button
            size="tiny"
            secondary
            type="error"
            @click.stop="emit('delete')"
          >
            <template #icon>
              <n-icon><TrashOutline /></n-icon>
            </template>
          </n-button>
        </template>
        Remove connection
      </n-tooltip>
    </div>
  </div>
</template>

<script setup>
import { FlashOutline, RefreshOutline, TrashOutline } from "@/lib/icons"

defineProps({
  /** Display name of the connection */
  label: {
    type: String,
    required: true,
  },
  /** True when the connection tree has already been fetched */
  hasTree: {
    type: Boolean,
    default: false,
  },
  /** True when this connection's result is currently active in the workspace */
  isActive: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(["connect", "delete", "dblclick"])
</script>
