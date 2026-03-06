import { Events } from '@wailsio/runtime'
import { ref } from 'vue'
// @ts-expect-error: generated bindings may not yet have typings
import { ListPlugins } from '@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager'

// shared reactive state; once initialized it stays in memory so every
// caller gets the same live list and loading state.  This mirrors the
// pattern used by useConnectionTree in the repo.
const plugins = ref<any[]>([])
const loading = ref(false)
const error = ref('')
let initialized = false

async function reload() {
  loading.value = true
  error.value = ''
  try {
    const list = await ListPlugins()
    plugins.value = Array.isArray(list) ? list : []
  }
  catch (err: any) {
    console.error('usePlugins.reload', err)
    error.value = err?.message ?? String(err)
    plugins.value = []
  }
  finally {
    loading.value = false
  }
}

function init() {
  if (initialized)
    return
  initialized = true
  // fetch immediately and listen for ready events
  reload()
  Events.On('plugins:ready', async () => {
    await reload()
  })
}

export function usePlugins() {
  init()
  return { plugins, loading, error, reload }
}
