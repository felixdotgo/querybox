<template>
  <div v-if="isMac" class="flex-shrink-0 h-[var(--macos-safe-zone)]" />
</template>

<script setup>
const isMac = navigator.userAgent.includes("Mac")
</script>
