<script setup>
import { computed, ref } from 'vue'

const props = defineProps({
  // The key label for this node (null for root document node)
  nodeKey: {
    type: String,
    default: null,
  },
  // The value to render (any JSON-compatible type)
  value: {
    required: true,
  },
  // Nesting depth — controls indentation (pl-4 per level when depth > 0)
  depth: {
    type: Number,
    default: 0,
  },
})

const PAGE_SIZE = 10

const isExpanded = ref(false)
const showAll = ref(false)

const valueType = computed(() => {
  if (props.value === null)
    return 'null'
  if (Array.isArray(props.value))
    return 'array'
  return typeof props.value
})

const isContainer = computed(() =>
  valueType.value === 'object' || valueType.value === 'array',
)

// Root document node: top-level object with no key — render children directly without a toggle row
const isRootDoc = computed(() =>
  props.nodeKey === null && props.depth === 0 && isContainer.value,
)

const entries = computed(() => {
  if (valueType.value === 'array')
    return props.value.map((v, i) => [String(i), v])
  if (valueType.value === 'object')
    return Object.entries(props.value)
  return []
})

const visibleEntries = computed(() =>
  showAll.value ? entries.value : entries.value.slice(0, PAGE_SIZE),
)

const hiddenCount = computed(() => Math.max(0, entries.value.length - PAGE_SIZE))

function toggle() {
  isExpanded.value = !isExpanded.value
}

function showMore() {
  showAll.value = true
}

function showLess() {
  showAll.value = false
}

// Inline primitive label
const primitiveLabel = computed(() => {
  const v = props.value
  if (v === null)
    return 'null'
  if (typeof v === 'string')
    return `"${v}"`
  return String(v)
})

const primitiveClass = computed(() => {
  if (props.value === null)
    return 'json-null'
  const t = typeof props.value
  if (t === 'string')
    return 'json-string'
  if (t === 'number')
    return 'json-number'
  if (t === 'boolean')
    return 'json-boolean'
  return ''
})

// Type label for container nodes: "Object" or "Array (n)"
const containerLabel = computed(() =>
  valueType.value === 'array'
    ? `Array (${entries.value.length})`
    : 'Object',
)

// "Show X more fields/items" text
const showMoreLabel = computed(() => {
  const n = hiddenCount.value
  return valueType.value === 'array'
    ? `Show ${n} more items`
    : `Show ${n} more fields`
})
</script>

<template>
  <!-- Root document node: render fields directly without a toggle row -->
  <template v-if="isRootDoc">
    <JsonNode
      v-for="[k, v] in visibleEntries"
      :key="k"
      :node-key="k"
      :value="v"
      :depth="0"
    />
    <div v-if="hiddenCount > 0" class="json-more-row">
      <button v-if="!showAll" class="json-more-btn" @click="showMore">
        <span class="json-more-arrow">▼</span>
        {{ showMoreLabel }}
      </button>
      <button v-else class="json-more-btn json-more-btn--less" @click="showLess">
        <span class="json-more-arrow">▲</span>
        Show less
      </button>
    </div>
  </template>

  <!-- Nested container node (object or array with a key label) -->
  <template v-else-if="isContainer">
    <div class="json-node" :class="[depth > 0 ? 'pl-4' : '']">
      <div class="json-row json-row-container" @click="toggle">
        <span class="json-triangle">{{ isExpanded ? '▼' : '▶' }}</span>
        <span class="json-key">{{ nodeKey }}</span>
        <span class="json-sep"> : </span>
        <span class="json-type-label">{{ containerLabel }}</span>
      </div>
      <div v-if="isExpanded">
        <JsonNode
          v-for="[k, v] in visibleEntries"
          :key="k"
          :node-key="k"
          :value="v"
          :depth="depth + 1"
        />
        <div v-if="hiddenCount > 0" class="json-more-row" :class="[depth > 0 ? 'pl-4' : '']">
          <button v-if="!showAll" class="json-more-btn" @click.stop="showMore">
            <span class="json-more-arrow">▼</span>
            {{ showMoreLabel }}
          </button>
          <button v-else class="json-more-btn json-more-btn--less" @click.stop="showLess">
            <span class="json-more-arrow">▲</span>
            Show less
          </button>
        </div>
      </div>
    </div>
  </template>

  <!-- Primitive node -->
  <template v-else>
    <div class="json-node" :class="[depth > 0 ? 'pl-4' : '']">
      <div class="json-row">
        <span class="json-triangle-spacer" aria-hidden="true" />
        <span v-if="nodeKey !== null" class="json-key">{{ nodeKey }}</span>
        <span v-if="nodeKey !== null" class="json-sep"> : </span>
        <span class="json-value" :class="primitiveClass">{{ primitiveLabel }}</span>
      </div>
    </div>
  </template>
</template>

<style scoped>
.json-node {
  font-family: var(--n-font-family-mono, monospace);
  font-size: 0.8125rem; /* 13px */
  line-height: 1.6;
}

.json-row {
  display: flex;
  align-items: baseline;
  gap: 0;
  min-height: 1.45rem;
  padding: 0 2px;
}

.json-row-container {
  cursor: pointer;
  user-select: none;
  border-radius: 2px;
}

.json-row-container:hover {
  background-color: rgba(0, 0, 0, 0.04);
}

/* Filled triangle indicator ▶ / ▼ */
.json-triangle {
  display: inline-block;
  width: 14px;
  font-size: 0.5rem;
  color: var(--n-text-color-3, #aaa);
  line-height: 1.6;
  flex-shrink: 0;
  text-align: center;
}

/* Invisible spacer to align keys on primitive rows with container rows */
.json-triangle-spacer {
  display: inline-block;
  width: 14px;
  flex-shrink: 0;
}

/* Key text */
.json-key {
  color: var(--n-text-color-2, #4b5563);
  font-weight: 500;
}

/* Space-colon-space separator */
.json-sep {
  color: var(--n-text-color-3, #9ca3af);
  white-space: pre;
}

/* "Object" / "Array (n)" type label */
.json-type-label {
  color: var(--n-text-color-3, #9ca3af);
}

/* Value colors by type */
.json-value {
  word-break: break-all;
}

.json-string {
  color: #16a34a; /* green-700 */
}

.json-number {
  color: #ea580c; /* orange-600 */
}

.json-boolean {
  color: #7c3aed; /* violet-700 */
}

.json-null {
  color: #7c3aed;
  font-style: italic;
}

/* "Show X more fields" row */
.json-more-row {
  padding: 1px 2px;
}

.json-more-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-family: inherit;
  font-size: 0.75rem;
  color: #0ea5e9; /* sky-500 */
  background: none;
  border: none;
  padding: 2px 0;
  cursor: pointer;
}

.json-more-btn:hover {
  color: #0284c7;
}

.json-more-btn--less {
  color: #6b7280;
}

.json-more-btn--less:hover {
  color: #374151;
}

.json-more-arrow {
  font-size: 0.5rem;
  line-height: 1;
}
</style>
