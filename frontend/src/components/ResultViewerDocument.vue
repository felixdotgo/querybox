<template>
  <pre class="whitespace-pre-wrap break-words text-sm">{{ formatted }}</pre>
</template>

<script setup>
import { computed } from "vue"

const props = defineProps({
  // Already-unwrapped document payload: { document: <any> }
  payload: {
    type: Object,
    required: true,
  },
})

const formatted = computed(() => {
  const doc = props.payload.document
  if (doc === null || doc === undefined) return ""
  if (typeof doc === "string") {
    try {
      return JSON.stringify(JSON.parse(doc), null, 2)
    } catch {
      return doc
    }
  }
  return JSON.stringify(doc, null, 2)
})
</script>
