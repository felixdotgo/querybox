<script setup>
import { NIcon } from 'naive-ui'
import { computed } from 'vue'
import { getDriverIcon } from '@/lib/dbIcons'
import { Server } from '@/lib/icons'

const props = defineProps({
  /** driver string as stored in `connection.driver_type` */
  driver: { type: String, required: true },
  /** desired icon size in pixels */
  size: { type: [Number, String], default: 14 },
  /** CSS colour to apply; defaults to `currentColor` so parent can style it */
  color: { type: String, default: 'currentColor' },
})

const svg = computed(() => {
  const si = getDriverIcon(props.driver)
  return si ? si.svg : ''
})
</script>

<template>
  <span
    v-if="svg"
    class="db-icon"
    :style="{ width: `${size}px`, height: `${size}px`, color }"
    v-html="svg"
  />
  <NIcon v-else :size="size">
    <template #default>
      <Server />
    </template>
  </NIcon>
</template>

<style scoped>
.db-icon svg {
  /* inherit the colour from the span so callers can simply set `color:` */
  fill: currentColor;
  width: 100%;
  height: 100%;
}
</style>
