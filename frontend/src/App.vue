<template>
  <n-config-provider>
    <n-message-provider>
      <router-view />
    </n-message-provider>
  </n-config-provider>
</template>

<script setup>
// n-config-provider is required for teleport-based components (n-dropdown,
// n-modal, n-message, etc.) to mount correctly.
</script>

<style scoped>
/* small scope for app-level adjustments if needed */
</style>
