<script setup>
import { computed } from 'vue'
import JsonNode from './JsonNode.vue'

const props = defineProps({
  // Already-unwrapped document payload: either
  //   { document: <any> }            (legacy/single-document)
  //   { documents: [<any>, ...] }    (current proto)
  payload: {
    type: Object,
    required: true,
  },
})

// Normalised list of document payloads — always an array
const docs = computed(() => {
  if (props.payload.documents !== undefined) {
    return Array.isArray(props.payload.documents)
      ? props.payload.documents
      : Array.from(props.payload.documents)
  }
  if (props.payload.document !== undefined) {
    return [props.payload.document]
  }
  return []
})
</script>

<template>
  <div class="h-full w-full overflow-auto p-2">
    <template v-if="docs.length">
      <div
        v-for="(doc, idx) in docs"
        :key="idx"
        class="doc-row"
      >
        <JsonNode :node-key="null" :value="doc" :depth="0" />
      </div>
    </template>
    <div v-else class="text-center text-gray-500 py-6 text-sm">
      (no documents)
    </div>
  </div>
</template>

<style scoped>
.doc-row {
  border: 1px solid var(--n-border-color, #e5e7eb);
  border-radius: 5px;
  margin-bottom: 6px;
  padding: 6px 10px;
  background-color: var(--n-color, #fff);
}
</style>
