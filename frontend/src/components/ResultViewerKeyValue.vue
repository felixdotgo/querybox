<template>
  <n-descriptions bordered column="1">
    <n-descriptions-item
      v-for="(v, k) in entries"
      :key="k"
      :label="String(k)"
    >
      {{ v }}
    </n-descriptions-item>
  </n-descriptions>
</template>

<script setup>
import { computed } from "vue"

const props = defineProps({
  // Already-unwrapped KV payload: { data: { key: value, ... } }
  payload: {
    type: Object,
    required: true,
  },
})

// Normalise: payload may be { data: {...} } or a flat object of k/v pairs.
const entries = computed(() => props.payload.data || props.payload || {})
</script>
