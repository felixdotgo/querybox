<script setup>
import { Events } from '@wailsio/runtime'
import { useNotification } from 'naive-ui'
import { computed, onMounted, ref, watch } from 'vue'
import { CloseConnectionsWindow } from '@/bindings/github.com/felixdotgo/querybox/services/app'
import { CreateConnection } from '@/bindings/github.com/felixdotgo/querybox/services/connectionservice'
import {
  TestConnection,
} from '@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager'
import { AuthFormRenderer } from '@/components/connections'
import DbIcon from '@/components/DbIcon.vue'
import { SafeZone } from '@/components/layout'
import { usePlugins } from '@/composables/usePlugins'
import { useAuthForms } from '@/composables/useAuthForms'
import { PluginType } from '@/lib/enums'

const notification = useNotification()

const { plugins, reload: reloadPlugins } = usePlugins()
const pluginFilter = ref('')
const selectedDriver = ref(null)
const statusText = ref('')
const testingConnection = ref(false)
const testResult = ref(null) // null | { ok: boolean, message: string }

// AuthForms state
const {
  authForms, selectedAuthForm, authValues,
  resetAuthState, loadAuthForms, serializeCredential,
} = useAuthForms()

const form = ref({ name: '', driver: '', cred: '' })

const drivers = computed(() => {
  // PluginInfo.type follows PluginV1.Type enum where DRIVER = 1
  // always return a sorted list so the UI is predictable
  return sortPlugins((plugins.value || []).filter(p => p && p.type === PluginType.DRIVER))
})

const filteredDrivers = computed(() => {
  const f = (pluginFilter.value || '').toLowerCase()
  return drivers.value.filter(
    p =>
      (p.name || '').toLowerCase().includes(f)
      || (p.description || '').toLowerCase().includes(f),
  )
})

// sort by name/id helper
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

const canConnect = computed(() => {
  // if auth forms are present for selected plugin, validate required fields
  const hasForms = Object.keys(authForms.value || {}).length > 0
  if (hasForms) {
    if (!selectedAuthForm.value)
      return false
    const formDef = authForms.value[selectedAuthForm.value]
    if (!formDef)
      return false
    for (const f of formDef.fields || []) {
      if (f.required && !(authValues.value[f.name] || '').toString().trim()) {
        return false
      }
    }
    return form.value.driver && form.value.name && form.value.name.trim()
  }

  return (
    form.value.driver
    && form.value.driver.trim()
    && form.value.cred
    && form.value.cred.trim()
    && form.value.name
    && form.value.name.trim()
  )
})

// watch plugin list for changes and react accordingly
watch(plugins, async (list) => {
  // reset search filter so new drivers become visible
  pluginFilter.value = ''
  if (!selectedDriver.value) {
    const firstDriver = (list || []).find(p => p && p.type === PluginType.DRIVER)
    if (firstDriver) {
      await selectPlugin(firstDriver)
    }
  }
}, { immediate: true })

async function selectPlugin(p) {
  selectedDriver.value = p
  form.value.driver = p.id || ''
  testResult.value = null

  // probe plugin for auth forms (graceful fallback to DSN input)
  await loadAuthForms(p.id)
}

function clearForm() {
  form.value = { name: '', driver: '', cred: '' }
  selectedDriver.value = null
  statusText.value = ''
  testResult.value = null
  pluginFilter.value = ''
}

async function testConnection() {
  if (!canConnect.value) {
    notification.warning({ title: 'Validation', content: 'Please select a driver and fill in all required fields', duration: 3000 })
    return
  }
  testingConnection.value = true
  testResult.value = null
  try {
    let cred = form.value.cred
    const serialized = serializeCredential()
    if (serialized) cred = serialized
    const params = { credential_blob: cred }
    const res = await TestConnection(form.value.driver.trim(), params)
    if (res) {
      testResult.value = { ok: res.ok, message: res.message || (res.ok ? 'Connection successful' : 'Connection failed') }
    }
    else {
      testResult.value = { ok: false, message: 'No response from plugin' }
    }
  }
  catch (err) {
    testResult.value = { ok: false, message: err?.message || String(err) }
  }
  finally {
    testingConnection.value = false
  }
}

async function saveConnection() {
  if (!canConnect.value) {
    notification.warning({ title: 'Validation', content: 'Please select a driver, provide a connection name and DSN', duration: 3000 })
    return
  }
  try {
    statusText.value = 'Connecting...'

    // if authForms in use, serialize the selected form values into credential_blob
    const serializedCred = serializeCredential()
    if (serializedCred) {
      form.value.cred = serializedCred
    }

    await CreateConnection(
      form.value.name.trim(),
      form.value.driver.trim(),
      form.value.cred.trim(),
    )
    // Backend emits connection:created — frontend only closes the window.
    await CloseConnectionsWindow()
  }
  catch (err) {
    console.error('connect:', err)
    statusText.value = 'Failed to save'
    notification.error({ title: 'Error', content: 'Failed to create connection', duration: 3000 })
  }
}

onMounted(async () => {
  // initial reload in case composable hasn't fetched yet (harmless duplicate)
  await reloadPlugins()

  // Clear form when window is closed (hidden)
  Events.On('connections-window:closed', async () => {
    clearForm()
    resetAuthState()

    // Re-select first driver if available
    const firstDriver = (plugins.value || []).find(p => p && p.type === PluginType.DRIVER)
    if (firstDriver) {
      await selectPlugin(firstDriver)
    }
  })
})
</script>

<template>
  <div class="flex flex-col h-screen">
    <div class="flex flex-1 min-h-0">
      <!-- Left: connection type list, fixed, no scroll -->
      <div class="w-52 shrink-0 border-r border-slate-200 overflow-y-auto bg-slate-50">
        <SafeZone />
        <div class="p-4">
          <h3 class="mb-2 font-bold">
            Connection Type
          </h3>
          <!-- simple filter box for long lists -->
          <div class="mb-2">
            <n-input
              v-model:value="pluginFilter"
              placeholder="filter drivers"
              size="small"
              clearable
              class="w-full"
            />
          </div>
          <ul class="list-none p-0 m-0 flex flex-col gap-1.5">
            <li v-if="drivers.length === 0" class="opacity-70">
              No drivers available
            </li>
            <li v-for="p in filteredDrivers" :key="p.id">
              <n-button
                block
                :type="
                  selectedDriver && selectedDriver.id === p.id
                    ? 'primary'
                    : 'default'
                "
                class="flex justify-between items-center text-left p-2"
                @click="selectPlugin(p)"
              >
                <div class="flex items-center gap-2">
                  <DbIcon :driver="(p.metadata?.simple_icon || p.id).toLowerCase()" size="16" />
                  <span>
                    {{ p.name }}
                    <small class="opacity-70 ml-1.5">{{ p.version || "" }}</small>
                  </span>
                </div>
              </n-button>
            </li>
          </ul>
        </div>
      </div>

      <!-- Right: connection detail form, scrolls independently -->
      <div class="flex-1 overflow-y-auto bg-white">
        <div class="p-4 pb-6">
          <div class="max-w-3xl">
            <div class="mb-4">
              <label class="block mb-1.5 text-gray-700 font-bold">Name</label>
              <n-input
                v-model:value="form.name"
                placeholder="e.g. local-mysql"
                class="w-full"
              />
            </div>

            <div>
              <n-tabs v-model:value="selectedAuthForm" type="card" class="mb-3">
                <n-tab-pane
                  v-for="(f, k) in authForms"
                  :key="k"
                  :name="k"
                  :tab="f.name"
                >
                  <AuthFormRenderer v-model="authValues" :form="f" />
                </n-tab-pane>
              </n-tabs>
            </div>

            <div>
              <n-input v-model:value="form.driver" readonly hidden />
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Bottom action bar, always visible -->
    <div class="shrink-0 p-4 bg-white border-t border-slate-200 shadow-sm">
      <n-flex justify="space-between" align="center">
        <n-button class="w-32" quaternary @click="CloseConnectionsWindow">
          Cancel
        </n-button>
        <n-flex align="center" class="gap-3">
          <n-button class="w-32" quaternary @click="clearForm">
            Reset
          </n-button>
          <n-button
            class="w-40"
            :disabled="!canConnect || testingConnection"
            :loading="testingConnection"
            @click="testConnection"
          >
            Test Connection
          </n-button>
          <n-button
            class="w-40"
            type="primary"
            :disabled="!canConnect"
            @click="saveConnection"
          >
            Save &amp; Connect
          </n-button>
        </n-flex>
      </n-flex>
      <div
        v-if="testResult"
        :class="testResult.ok ? 'text-green-600' : 'text-red-500'"
        class="mt-2 text-sm text-right"
      >
        {{ testResult.ok ? '✓' : '✗' }} {{ testResult.message }}
      </div>
    </div>
  </div>
</template>
