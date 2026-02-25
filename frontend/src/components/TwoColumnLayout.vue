<script setup>
import { ref } from 'vue'

defineEmits(['dragstart'])

// expose container element ref so parent can use it for resizing logic
const containerRef = ref(null)
defineExpose({ containerRef })
</script>

<template>
  <div ref="containerRef" class="flex w-full h-full overflow-hidden">
    <div
      class="flex-shrink-0 h-full overflow-hidden"
      :style="{ width: `${leftWidth}px`, minWidth: `${minLeftWidth}px` }"
    >
      <slot name="left" />
    </div>

    <div
      role="separator"
      aria-orientation="vertical"
      class="w-1 cursor-col-resize bg-gray-200 hover:bg-sky-500"
      @pointerdown="$emit('dragstart', $event)"
    >
      <div class="w-0.5 h-full bg-transparent mx-auto" />
    </div>

    <div class="flex-1 min-h-0 overflow-auto flex flex-col">
      <slot name="right" />
    </div>
  </div>
</template>
