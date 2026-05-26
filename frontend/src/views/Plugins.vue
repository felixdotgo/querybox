<script setup>
import { Call, Events } from '@wailsio/runtime'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ClosePluginsWindow, OpenURL } from '@/bindings/github.com/felixdotgo/querybox/services/app'
import { Rescan } from '@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager'
import { SafeZone } from '@/components/layout'
import { usePlugins } from '@/composables/usePlugins'
import { PLUGIN_TYPE_LABELS } from '@/lib/enums'

const { plugins, reload: reloadPlugins } = usePlugins()
const filter = ref('')
const loading = ref(false)
const loadError = ref('')
const selected = ref(null)
const updateStatus = ref(null)

function semverGreater(a, b) {
  const parse = v => (v || '').split('.').map(Number)
  const [am, ai, ap] = parse(a)
  const [bm, bi, bp] = parse(b)
  if (am !== bm)
    return am > bm
  if (ai !== bi)
    return ai > bi
  return ap > bp
}

function pluginHasUpdate(plugin) {
  if (!updateStatus.value?.pluginRegistry)
    return false
  const entry = updateStatus.value.pluginRegistry[plugin.name?.toLowerCase()]
  if (!entry)
    return false
  return semverGreater(entry.version, plugin.version)
}

function pluginLatestVersion(plugin) {
  return updateStatus.value?.pluginRegistry?.[plugin.name?.toLowerCase()]?.version
}

function pluginReleaseURL(plugin) {
  const v = pluginLatestVersion(plugin)
  if (!v)
    return ''
  return `https://github.com/felixdotgo/querybox/releases/tag/plugin-${plugin.name?.toLowerCase()}-v${v}`
}

// keep the off-function so we can deregister on unmount
let offPluginsOpened = null

const filteredPlugins = computed(() => {
  const f = filter.value.toLowerCase()
  if (!f)
    return plugins.value
  return plugins.value.filter(
    p =>
      (p.name || '').toLowerCase().includes(f)
      || (p.description || '').toLowerCase().includes(f),
  )
})

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    await Rescan()
    await reloadPlugins()
    // keep selection in sync after reload
    if (selected.value) {
      selected.value = plugins.value.find(p => p.id === selected.value.id) ?? null
    }
    if (!selected.value && plugins.value.length > 0) {
      selected.value = plugins.value[0]
    }
  }
  catch (err) {
    console.error('load plugins:', err)
    loadError.value = err?.message ?? String(err)
  }
  finally {
    loading.value = false
  }
}

onMounted(async () => {
  await load()
  offPluginsOpened = Events.On('plugins-window:opened', load)
  window.addEventListener('focus', load)
  Call.ByName('updater.Updater.GetUpdateStatus')
    .then((s) => { updateStatus.value = s })
    .catch(() => {})
})

onUnmounted(() => {
  window.removeEventListener('focus', load)
  if (typeof offPluginsOpened === 'function')
    offPluginsOpened()
})

function handleClose() {
  // Just hide the window — never navigate away, or the webview will show the
  // wrong route the next time ShowPluginsWindow() is called from the backend.
  ClosePluginsWindow().catch(err => console.warn('ClosePluginsWindow:', err))
}

function typeLabel(type) {
  return PLUGIN_TYPE_LABELS[type] || (type ? `Type ${type}` : '—')
}

function mapEntries(obj) {
  if (!obj || typeof obj !== 'object')
    return []
  return Object.entries(obj)
}

function formatRuntime(runtime) {
  if (!runtime)
    return '—'
  const parts = [runtime.kind || 'unknown']
  if (runtime.entrypoint) {
    parts.push(runtime.entrypoint)
  }
  if (Array.isArray(runtime.args) && runtime.args.length) {
    parts.push(runtime.args.join(' '))
  }
  return parts.join(' · ')
}

function formatPermissions(permissions) {
  if (!Array.isArray(permissions) || permissions.length === 0)
    return '—'
  return permissions
    .map((permission) => {
      const name = permission?.name || 'unnamed'
      return permission?.required ? `${name} (required)` : name
    })
    .join(', ')
}

function limitRows(limits) {
  if (!limits || typeof limits !== 'object')
    return []
  const rows = []
  if (limits.timeout_seconds)
    rows.push(['Timeout', `${limits.timeout_seconds}s`])
  if (limits.max_output_bytes)
    rows.push(['Max output', `${limits.max_output_bytes} bytes`])
  if (limits.working_dir)
    rows.push(['Working dir', limits.working_dir])
  if (Array.isArray(limits.env_allowlist) && limits.env_allowlist.length)
    rows.push(['Env allowlist', limits.env_allowlist.join(', ')])
  return rows
}
</script>

<template>
  <div class="h-screen flex flex-col bg-white font-mono text-sm">
    <SafeZone />

    <!-- Top bar -->
    <div class="shrink-0 flex items-center justify-between px-4 py-2.5 border-b border-slate-200">
      <span class="font-semibold text-slate-700">Installed Plugins</span>
      <n-button size="small" quaternary :loading="loading" @click="load">
        Refresh
      </n-button>
    </div>

    <!-- App update banner -->
    <div v-if="updateStatus?.appUpdateAvailable" class="shrink-0 text-xs text-amber-800 bg-amber-50 border-b border-amber-200 px-4 py-2 flex items-center justify-between">
      <span>QueryBox {{ updateStatus.appLatestVersion }} is available (current: {{ updateStatus.appCurrentVersion }})</span>
      <button class="ml-4 underline hover:no-underline" @click="OpenURL(updateStatus.appReleaseUrl)">
        Download →
      </button>
    </div>

    <!-- Error banner -->
    <div v-if="loadError" class="shrink-0 text-xs text-red-700 bg-red-50 border-b border-red-200 px-4 py-2 flex justify-between">
      <span>{{ loadError }}</span>
      <span class="cursor-pointer underline ml-4" @click="loadError = ''">dismiss</span>
    </div>

    <!-- Main two-column body -->
    <div class="flex-1 flex overflow-hidden">
      <!-- Left: plugin list -->
      <div class="w-56 shrink-0 flex flex-col border-r border-slate-200 overflow-hidden">
        <!-- Search -->
        <div class="px-2 py-2 border-b border-slate-100">
          <n-input
            v-model:value="filter"
            placeholder="Search…"
            clearable
            size="small"
          />
        </div>

        <!-- List -->
        <div class="flex-1 overflow-y-auto">
          <!-- Loading -->
          <div v-if="loading" class="text-xs text-slate-400 text-center mt-6">
            Loading…
          </div>

          <!-- Empty -->
          <div v-else-if="filteredPlugins.length === 0" class="text-xs text-slate-400 text-center mt-6 px-3">
            No plugins found
          </div>

          <!-- Items -->
          <button
            v-for="p in filteredPlugins"
            :key="p.id || p.name"
            class="w-full text-left px-3 py-2.5 border-b border-slate-100 hover:bg-slate-50 transition-colors"
            :class="selected?.id === p.id ? 'bg-blue-50 border-l-2 border-l-blue-500' : 'border-l-2 border-l-transparent'"
            @click="selected = p"
          >
            <div class="flex items-center gap-1.5">
              <span class="font-medium text-slate-800 truncate">{{ p.name || p.id }}</span>
              <span v-if="pluginHasUpdate(p)" class="w-1.5 h-1.5 rounded-full bg-amber-400 shrink-0" title="Update available" />
            </div>
            <div class="text-xs text-slate-400 truncate mt-0.5">
              {{ p.version ? `v${p.version}` : '' }}
              <span v-if="p.version && p.author"> · </span>
              {{ p.author || '' }}
            </div>
          </button>
        </div>
      </div>

      <!-- Right: detail panel -->
      <div class="flex-1 overflow-y-auto p-6">
        <!-- No selection -->
        <div v-if="!selected" class="text-slate-400 text-xs mt-8 text-center">
          Select a plugin to view details
        </div>

        <!-- Detail -->
        <template v-else>
          <!-- Title row -->
          <div class="flex items-start justify-between gap-4 mb-5">
            <div class="flex items-start gap-3 min-w-0">
              <img
                v-if="selected.icon_url"
                :src="selected.icon_url"
                :alt="`${selected.name || selected.id} icon`"
                class="w-10 h-10 rounded border border-slate-200 object-contain bg-white p-1 shrink-0"
              >
              <div class="min-w-0">
                <h2 class="text-base font-semibold text-slate-800">
                  {{ selected.name || selected.id }}
                </h2>
                <div class="text-xs text-slate-400 mt-0.5">
                  {{ selected.id }}
                </div>
              </div>
            </div>
            <span
              v-if="selected.type"
              class="shrink-0 text-xs px-2 py-0.5 rounded-full bg-blue-100 text-blue-700 font-medium"
            >
              {{ typeLabel(selected.type) }}
            </span>
          </div>

          <!-- Description -->
          <p v-if="selected.description" class="text-slate-600 text-xs leading-relaxed mb-5">
            {{ selected.description }}
          </p>

          <!-- Update available notice -->
          <div v-if="pluginHasUpdate(selected)" class="mb-5 text-xs bg-amber-50 border border-amber-200 rounded px-3 py-2 flex items-center justify-between">
            <span class="text-amber-800">Version {{ pluginLatestVersion(selected) }} available</span>
            <button class="ml-4 underline text-amber-700 hover:no-underline" @click="OpenURL(pluginReleaseURL(selected))">
              View release →
            </button>
          </div>

          <!-- Key/value grid -->
          <div class="grid grid-cols-[120px_1fr] gap-x-4 gap-y-2 text-xs">
            <template v-if="selected.version">
              <span class="text-slate-400">Version</span>
              <span class="text-slate-700">{{ selected.version }}</span>
            </template>
            <template v-if="selected.author">
              <span class="text-slate-400">Author</span>
              <span class="text-slate-700">{{ selected.author }}</span>
            </template>
            <template v-if="selected.license">
              <span class="text-slate-400">License</span>
              <span class="text-slate-700">{{ selected.license }}</span>
            </template>
            <template v-if="selected.url">
              <span class="text-slate-400">URL</span>
              <a :href="selected.url" target="_blank" class="text-blue-600 hover:underline truncate">{{ selected.url }}</a>
            </template>
            <template v-if="selected.contact">
              <span class="text-slate-400">Contact</span>
              <span class="text-slate-700">{{ selected.contact }}</span>
            </template>
            <template v-if="selected.path">
              <span class="text-slate-400">Path</span>
              <span class="text-slate-500 font-mono break-all">{{ selected.path }}</span>
            </template>
            <template v-if="selected.manifest_path">
              <span class="text-slate-400">Manifest</span>
              <span class="text-slate-500 font-mono break-all">{{ selected.manifest_path }}</span>
            </template>
            <template v-if="selected.runtime">
              <span class="text-slate-400">Runtime</span>
              <span class="text-slate-700">{{ formatRuntime(selected.runtime) }}</span>
            </template>
            <template v-if="selected.permissions?.length">
              <span class="text-slate-400">Permissions</span>
              <span class="text-slate-700">{{ formatPermissions(selected.permissions) }}</span>
            </template>
          </div>

          <!-- Capabilities -->
          <div v-if="selected.capabilities?.length" class="mt-4">
            <div class="text-xs text-slate-400 mb-1.5">
              Capabilities
            </div>
            <div class="flex flex-wrap gap-1.5">
              <span
                v-for="capability in selected.capabilities"
                :key="capability"
                class="text-xs px-2 py-0.5 rounded bg-blue-50 text-blue-700 border border-blue-100"
              >{{ capability }}</span>
            </div>
          </div>

          <!-- Tags -->
          <div v-if="selected.tags?.length" class="mt-4">
            <div class="text-xs text-slate-400 mb-1.5">
              Tags
            </div>
            <div class="flex flex-wrap gap-1.5">
              <span
                v-for="tag in selected.tags"
                :key="tag"
                class="text-xs px-2 py-0.5 rounded bg-slate-100 text-slate-600"
              >{{ tag }}</span>
            </div>
          </div>

          <!-- Limits -->
          <div v-if="limitRows(selected.limits).length" class="mt-4">
            <div class="text-xs text-slate-400 mb-1.5">
              Limits
            </div>
            <div class="grid grid-cols-[120px_1fr] gap-x-4 gap-y-2 text-xs">
              <template v-for="[label, value] in limitRows(selected.limits)" :key="label">
                <span class="text-slate-400">{{ label }}</span>
                <span class="text-slate-700 break-all">{{ value }}</span>
              </template>
            </div>
          </div>

          <!-- Metadata -->
          <div v-if="mapEntries(selected.metadata).length" class="mt-4">
            <div class="text-xs text-slate-400 mb-1.5">
              Metadata
            </div>
            <div class="grid grid-cols-[160px_1fr] gap-x-4 gap-y-2 text-xs">
              <template v-for="[key, value] in mapEntries(selected.metadata)" :key="key">
                <span class="text-slate-400 font-mono">{{ key }}</span>
                <span class="text-slate-700 break-all">{{ value }}</span>
              </template>
            </div>
          </div>

          <!-- Settings -->
          <div v-if="mapEntries(selected.settings).length" class="mt-4">
            <div class="text-xs text-slate-400 mb-1.5">
              Settings
            </div>
            <div class="grid grid-cols-[160px_1fr] gap-x-4 gap-y-2 text-xs">
              <template v-for="[key, value] in mapEntries(selected.settings)" :key="key">
                <span class="text-slate-400 font-mono">{{ key }}</span>
                <span class="text-slate-700 break-all">{{ value }}</span>
              </template>
            </div>
          </div>

          <!-- Error -->
          <div v-if="selected.lastError" class="mt-5 text-xs text-red-600 bg-red-50 border border-red-200 rounded px-3 py-2">
            <span class="font-medium">Error:</span> {{ selected.lastError }}
          </div>
        </template>
      </div>
    </div>

    <!-- Footer -->
    <div class="shrink-0 px-4 py-2.5 border-t border-slate-200">
      <n-button size="small" quaternary @click="handleClose">
        Close
      </n-button>
    </div>
  </div>
</template>
