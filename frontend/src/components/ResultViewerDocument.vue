<template>
  <div class="h-full w-full overflow-auto p-2">
    <template v-if="docs.length">
      <n-space vertical size="small" class="w-full">
        <n-card
          v-for="(doc, idx) in docs"
          :key="idx"
          bordered
          size="small"
          class="w-full"
        >
          <pre class="whitespace-pre-wrap break-words text-sm rounded bg-gray-50 p-2"><code v-html="highlight(doc)"></code></pre>
        </n-card>
      </n-space>
    </template>
    <div v-else class="text-center text-gray-500">(no documents)</div>
  </div>
</template>

<script setup>
import { computed } from "vue"
import hljs from "highlight.js/lib/core"
import jsonLang from "highlight.js/lib/languages/json"

// register only json to keep bundle small
hljs.registerLanguage("json", jsonLang)

const props = defineProps({
  // Already-unwrapped document payload: either
  //   { document: <any> }            (legacy/single-document)
  //   { documents: [<any>, ...] }    (current proto)
  payload: {
    type: Object,
    required: true,
  },
})

// Normalised list of document payloads. always an array
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

/**
 * Return a prettified string representation of a single document.
 */
function format(doc) {
  if (doc === null || doc === undefined) return ""
  if (typeof doc === "string") {
    try {
      return JSON.stringify(JSON.parse(doc), null, 2)
    } catch {
      return doc
    }
  }
  return JSON.stringify(doc, null, 2)
}

/**
 * Return highlighted HTML using highlight.js.  relies on JSON language.
 */
function highlight(doc) {
  const code = format(doc)
  const { value } = hljs.highlight(code, { language: "json" })
  return value
}
</script>
