<script setup>
import { Events } from '@wailsio/runtime'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ClosePluginsWindow } from '@/bindings/github.com/felixdotgo/querybox/services/app'
import { ListPlugins, Rescan } from '@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager'
import SafeZone from '@/components/SafeZone.vue'

const TYPE_LABELS = { 1: 'Driver', 2: 'Transformer', 3: 'Formatter' }

const plugins = ref([])
const filter = ref('')
const loading = ref(false)
const loadError = ref('')
const selected = ref(null)

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
    const plist = await ListPlugins()
    // JSON round-trip converts class instances to plain objects so Vue's
    // reactivity proxy doesn't corrupt the VNode update cycle.
    plugins.value = JSON.parse(JSON.stringify(sortPlugins(plist ?? [])))
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
    plugins.value = []
  }
  finally {
    loading.value = false
  }
}

onMounted(async () => {
  await load()
  offPluginsOpened = Events.On('plugins-window:opened', load)
  window.addEventListener('focus', load)
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
  return TYPE_LABELS[type] || (type ? `Type ${type}` : '—')
}

// sort by name case-insensitively, fall back to id when name is empty
function sortPlugins(list) {
  return list.slice().sort((a, b) => {
    const aName = (a.name || a.id || '').toLowerCase()
    const bName = (b.name || b.id || '').toLowerCase()
    if (aName < bName)
      return -1
    if (aName > bName)
      return 1
    return 0
  })
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
            <div class="font-medium text-slate-800 truncate">
              {{ p.name || p.id }}
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
            <div>
              <h2 class="text-base font-semibold text-slate-800">
                {{ selected.name || selected.id }}
              </h2>
              <div class="text-xs text-slate-400 mt-0.5">
                {{ selected.id }}
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
