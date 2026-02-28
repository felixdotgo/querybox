<script setup>
import { NIcon } from 'naive-ui'
import { computed, h } from 'vue'
import { EllipsisHorizontal, Flash, Refresh, Trash } from '@/lib/icons'

const props = defineProps({
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
  /** Show a loading indicator on the connect button */
  loading: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['connect', 'delete', 'dblclick'])

function renderIcon(icon) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

const menuOptions = computed(() => [
  {
    key: 'connect',
    label: props.hasTree ? 'Reconnect' : 'Connect',
    icon: renderIcon(props.hasTree ? Refresh : Flash),
    disabled: props.loading,
  },
  { type: 'divider', key: 'divider-1' },
  {
    key: 'delete',
    label: 'Remove connection',
    icon: renderIcon(Trash),
  },
])

function handleMenuSelect(key) {
  if (key === 'connect')
    emit('connect')
  else if (key === 'delete')
    emit('delete')
}
</script>

<template>
  <div
    class="flex items-center justify-between w-full group/conn pr-1"
    @dblclick.stop="emit('dblclick')"
  >
    <!-- connection name -->
    <n-ellipsis class="flex-1 min-w-0 text-sm">
      {{ label }}
    </n-ellipsis>

    <!-- three-dot context menu — revealed on hover via CSS group -->
    <div class="hidden group-hover/conn:flex flex-shrink-0 ml-1">
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
