<template>
  <div class="flex flex-col h-screen">
    <div class="flex flex-1 min-h-0">
      <!-- Left: connection type list, fixed, no scroll -->
      <div class="w-52 shrink-0 border-r border-slate-200 overflow-y-auto bg-slate-50">
        <SafeZone />
        <div class="p-4">
          <h3 class="mb-2 font-bold">Connection Type</h3>
          <ul class="list-none p-0 m-0 flex flex-col gap-1.5">
            <li v-if="drivers.length === 0" class="opacity-70">
              No drivers available
            </li>
            <li v-for="p in filteredDrivers" :key="p.name">
              <n-button
                block
                :type="
                  selectedDriver && selectedDriver.name === p.name
                    ? 'primary'
                    : 'default'
                "
                @click="selectPlugin(p)"
                class="flex justify-between items-center text-left p-2"
              >
                <div>
                  {{ p.name }}
                  <small class="opacity-70 ml-1.5">{{ p.version || "" }}</small>
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
              <n-tabs type="card" v-model:value="selectedAuthForm" class="mb-3">
                <n-tab-pane
                  v-for="(f, k) in authForms"
                  :key="k"
                  :name="k"
                  :tab="f.name"
                >
                  <AuthFormRenderer :form="f" v-model="authValues" />
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
          <n-button class="w-32" quaternary @click="clearForm">Reset</n-button>
          <n-button
            class="w-40"
            @click="testConnection"
            :disabled="!canConnect || testingConnection"
            :loading="testingConnection"
          >
            Test Connection
          </n-button>
          <n-button
            class="w-40"
            type="primary"
            @click="saveConnection"
            :disabled="!canConnect"
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

<script setup>
import { ref, computed, onMounted, watch } from "vue"
import {
  ListPlugins,
  GetPluginAuthForms,
  TestConnection,
} from "@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager"
import { CloseConnectionsWindow } from "@/bindings/github.com/felixdotgo/querybox/services/app"
import { CreateConnection } from "@/bindings/github.com/felixdotgo/querybox/services/connectionservice"
import AuthFormRenderer from "@/components/AuthFormRenderer.vue"
import SafeZone from "@/components/SafeZone.vue"

const plugins = ref([])
const pluginFilter = ref("")
const selectedDriver = ref(null)
const statusText = ref("")
const testingConnection = ref(false)
const testResult = ref(null) // null | { ok: boolean, message: string }

// AuthForms state
const authForms = ref({})
const selectedAuthForm = ref("")
const authValues = ref({})

const form = ref({ name: "", driver: "", cred: "" })
const filterText = ref("")

function resetAuthState() {
  authForms.value = {}
  selectedAuthForm.value = ""
  authValues.value = {}
}

const drivers = computed(() => {
  // PluginInfo.type follows PluginV1.Type enum where DRIVER = 1
  return (plugins.value || []).filter((p) => p && p.type === 1)
})

const filteredDrivers = computed(() => {
  const f = (pluginFilter.value || "").toLowerCase()
  return drivers.value.filter(
    (p) =>
      (p.name || "").toLowerCase().includes(f) ||
      (p.description || "").toLowerCase().includes(f),
  )
})

const canConnect = computed(() => {
  // if auth forms are present for selected plugin, validate required fields
  const hasForms = Object.keys(authForms.value || {}).length > 0
  if (hasForms) {
    if (!selectedAuthForm.value) return false
    const formDef = authForms.value[selectedAuthForm.value]
    if (!formDef) return false
    for (const f of formDef.fields || []) {
      if (f.required && !(authValues.value[f.name] || "").toString().trim()) {
        return false
      }
    }
    return form.value.driver && form.value.name && form.value.name.trim()
  }

  return (
    form.value.driver &&
    form.value.driver.trim() &&
    form.value.cred &&
    form.value.cred.trim() &&
    form.value.name &&
    form.value.name.trim()
  )
})

async function load() {
  try {
    const [plist] = await Promise.all([ListPlugins()])
    plugins.value = plist || []

    // Auto-select the first available driver by default when opening the
    // Connections view and nothing is currently selected.
    if (!selectedDriver.value) {
      const firstDriver = (plugins.value || []).find((p) => p && p.type === 1)
      if (firstDriver) {
        // use selectPlugin to initialize auth forms and defaults
        await selectPlugin(firstDriver)
      }
    }
  } catch (err) {
    console.error("load:", err)
    plugins.value = plugins.value || []
  }
}

async function selectPlugin(p) {
  selectedDriver.value = p
  form.value.driver = p.name || ""
  testResult.value = null

  // probe plugin for auth forms (graceful fallback to DSN input)
  resetAuthState()
  try {
    const resp = await GetPluginAuthForms(p.name)
    if (resp && Object.keys(resp).length > 0) {
      authForms.value = resp || {}
      const keys = Object.keys(authForms.value)
      selectedAuthForm.value = keys[0]
      // initialize values object for selected form
      authValues.value = {}
      for (const f of authForms.value[selectedAuthForm.value].fields || []) {
        authValues.value[f.name] = f.value || ""
      }
      // pre-fill credential field with serialized blob for convenience (not required)
      // leave `form.cred` empty — CreateConnection will serialize current form values
      return
    }
  } catch (err) {
    console.error("GetPluginAuthForms:", err)
  }
}

// Keep authValues in sync when the user switches auth form tabs.
// Only initialise fields that have no value yet — this preserves values
// the user already typed (common fields between tabs) and correctly
// applies per-field defaults without wiping the parent's manual init.
watch(selectedAuthForm, (newKey) => {
  if (!newKey) return
  const def = authForms.value[newKey]
  if (!def) return
  for (const f of def.fields || []) {
    if (authValues.value[f.name] === undefined || authValues.value[f.name] === null) {
      authValues.value[f.name] = f.value ?? ""
    }
  }
})

function clearForm() {
  form.value = { name: "", driver: "", cred: "" }
  selectedDriver.value = null
  statusText.value = ""
  testResult.value = null
}

async function testConnection() {
  if (!canConnect.value) {
    alert("Please select a driver and fill in all required fields")
    return
  }
  testingConnection.value = true
  testResult.value = null
  try {
    let cred = form.value.cred
    if (Object.keys(authForms.value || {}).length > 0) {
      const blob = { form: selectedAuthForm.value, values: authValues.value }
      cred = JSON.stringify(blob)
    }
    const params = { credential_blob: cred }
    const res = await TestConnection(form.value.driver.trim(), params)
    if (res) {
      testResult.value = { ok: res.ok, message: res.message || (res.ok ? "Connection successful" : "Connection failed") }
    } else {
      testResult.value = { ok: false, message: "No response from plugin" }
    }
  } catch (err) {
    testResult.value = { ok: false, message: err?.message || String(err) }
  } finally {
    testingConnection.value = false
  }
}

async function saveConnection() {
  if (!canConnect.value) {
    alert("Please select a driver, provide a connection name and DSN")
    return
  }
  try {
    statusText.value = "Connecting..."

    // if authForms in use, serialize the selected form values into credential_blob
    if (Object.keys(authForms.value || {}).length > 0) {
      const blob = { form: selectedAuthForm.value, values: authValues.value }
      form.value.cred = JSON.stringify(blob)
    }

    await CreateConnection(
      form.value.name.trim(),
      form.value.driver.trim(),
      form.value.cred.trim(),
    )
    // Backend emits connection:created — frontend only closes the window.
    await CloseConnectionsWindow()
  } catch (err) {
    console.error("connect:", err)
    statusText.value = "Failed to save"
    alert("Failed to create connection")
  }
}

onMounted(async () => {
  await load()
})
</script>
