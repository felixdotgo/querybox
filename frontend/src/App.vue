<script setup>
// n-config-provider is required for teleport-based components (n-dropdown,
// n-modal, n-message, etc.) to mount correctly.
//
// themeOverrides enforce JetBrains Mono across all Naive UI components —
// unified monospace type for a developer-tool aesthetic.
const themeOverrides = {
  common: {
    fontFamily:
      '"JetBrains Mono", ui-monospace, monospace',
    fontFamilyMono:
      '"JetBrains Mono", ui-monospace, monospace',
  },
}
</script>

<template>
  <n-config-provider :theme-overrides="themeOverrides">
    <n-dialog-provider>
      <n-message-provider>
        <n-notification-provider>
          <router-view />
        </n-notification-provider>
      </n-message-provider>
    </n-dialog-provider>
  </n-config-provider>
</template>

<style scoped>
/* small scope for app-level adjustments if needed */
</style>
