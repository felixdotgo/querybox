<template>
  <n-config-provider :theme-overrides="themeOverrides">
    <n-message-provider>
      <router-view />
    </n-message-provider>
  </n-config-provider>
</template>

<script setup>
// n-config-provider is required for teleport-based components (n-dropdown,
// n-modal, n-message, etc.) to mount correctly.
//
// themeOverrides enforce Inter (UI) + Fira Code (mono) across all Naive UI
// components, matching the developer-focused aesthetic of tools like MongoDB Compass.
const themeOverrides = {
  common: {
    fontFamily:
      '"Inter", ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, sans-serif',
    fontFamilyMono:
      '"Fira Code", "Cascadia Code", "JetBrains Mono", ui-monospace, monospace',
  },
};
</script>

<style scoped>
/* small scope for app-level adjustments if needed */
</style>
