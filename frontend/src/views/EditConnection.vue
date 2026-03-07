<script setup>
import { Events } from '@wailsio/runtime'
import { useNotification } from 'naive-ui'
import { computed, onMounted, ref, watch } from 'vue'
import { CloseEditConnectionWindow } from '@/bindings/github.com/felixdotgo/querybox/services/app'
import {
  GetConnection,
  GetCredential,
  UpdateConnection,
} from '@/bindings/github.com/felixdotgo/querybox/services/connectionservice'
import {
  GetPluginAuthForms,
  TestConnection,
} from '@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager'
import AuthFormRenderer from '@/components/AuthFormRenderer.vue'
import SafeZone from '@/components/SafeZone.vue'

const notification = useNotification()

const connectionId = ref('')
const connectionDriverType = ref('')
const connectionDriverName = ref('')
const form = ref({ name: '' })

const authForms = ref({})
const selectedAuthForm = ref('')
const authValues = ref({})
const rawCred = ref('')

const testResult = ref(null)
const testingConnection = ref(false)
const saving = ref(false)

function resetForm() {
  connectionId.value = ''
  connectionDriverType.value = ''
  connectionDriverName.value = ''
  form.value = { name: '' }
  authForms.value = {}
  selectedAuthForm.value = ''
  authValues.value = {}
  rawCred.value = ''
  testResult.value = null
}

const canSave = computed(() => {
  if (!form.value.name || !form.value.name.trim())
    return false
  const hasForms = Object.keys(authForms.value || {}).length > 0
  if (hasForms) {
    if (!selectedAuthForm.value)
      return false
    const formDef = authForms.value[selectedAuthForm.value]
    if (!formDef)
      return false
    for (const f of formDef.fields || []) {
      if (f.required && !(authValues.value[f.name] || '').toString().trim())
        return false
    }
  }
  return true
})

// Keep authValues in sync when the user switches auth form tabs.
watch(selectedAuthForm, (newKey) => {
  if (!newKey)
    return
  const def = authForms.value[newKey]
  if (!def)
    return
  for (const f of def.fields || []) {
    if (authValues.value[f.name] === undefined || authValues.value[f.name] === null)
      authValues.value[f.name] = f.value ?? ''
  }
})

async function loadConnection(id) {
  try {
    // Fetch connection metadata, its auth form definition, and its stored credential in parallel.
    const [conn, cred] = await Promise.all([
      GetConnection(id),
      GetCredential(id),
    ])

    connectionId.value = conn.id
    connectionDriverType.value = conn.driver_type
    connectionDriverName.value = conn.driver_type
    form.value = { name: conn.name }

    // Load auth forms for this driver
    try {
      const resp = await GetPluginAuthForms(conn.driver_type)
      if (resp && Object.keys(resp).length > 0) {
        authForms.value = resp
        // Parse existing credential blob to pre-fill form
        let parsedForm = ''
        let parsedValues = {}
        try {
          const blob = JSON.parse(cred)
          parsedForm = blob.form || ''
          parsedValues = blob.values || {}
        }
        catch {
          rawCred.value = cred
        }

        const formKeys = Object.keys(authForms.value)
        // Use the saved form tab if it still exists, otherwise fall back to first tab
        selectedAuthForm.value = (parsedForm && authForms.value[parsedForm]) ? parsedForm : formKeys[0]

        // Initialize all fields with defaults then overwrite with saved values
        authValues.value = {}
        for (const f of authForms.value[selectedAuthForm.value]?.fields || []) {
          authValues.value[f.name] = f.value ?? ''
        }
        Object.assign(authValues.value, parsedValues)
      }
      else {
        rawCred.value = cred
      }
    }
    catch (err) {
      console.debug('GetPluginAuthForms (edit, ignored):', err)
      rawCred.value = cred
    }
  }
  catch (err) {
    console.error('loadConnection:', err)
    notification.error({ title: 'Error', content: 'Failed to load connection details', duration: 3000 })
  }
}

async function testConnection() {
  if (!canSave.value) {
    notification.warning({ title: 'Validation', content: 'Please fill in all required fields', duration: 3000 })
    return
  }
  testingConnection.value = true
  testResult.value = null
  try {
    let cred = rawCred.value
    if (Object.keys(authForms.value || {}).length > 0) {
      cred = JSON.stringify({ form: selectedAuthForm.value, values: authValues.value })
    }
    const res = await TestConnection(connectionDriverType.value, { credential_blob: cred })
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
  if (!canSave.value) {
    notification.warning({ title: 'Validation', content: 'Please fill in all required fields', duration: 3000 })
    return
  }
  saving.value = true
  try {
    let cred = rawCred.value
    if (Object.keys(authForms.value || {}).length > 0) {
      cred = JSON.stringify({ form: selectedAuthForm.value, values: authValues.value })
    }
    await UpdateConnection(connectionId.value, form.value.name.trim(), cred)
    await CloseEditConnectionWindow()
  }
  catch (err) {
    console.error('saveConnection:', err)
    notification.error({ title: 'Error', content: 'Failed to save connection', duration: 3000 })
  }
  finally {
    saving.value = false
  }
}

onMounted(() => {
  Events.On('edit-connection-window:opened', (event) => {
    const id = event?.data?.id
    if (id)
      loadConnection(id)
  })

  Events.On('edit-connection-window:closed', () => {
    resetForm()
  })
})
</script>

<template>
  <div class="flex flex-col h-screen">
    <div class="flex flex-1 min-h-0">
      <!-- Main form area -->
      <div class="flex-1 overflow-y-auto bg-white">
        <SafeZone />
        <div class="p-4 pb-6">
          <div class="max-w-3xl">

            <!-- Connection name -->
            <div class="mb-4">
              <label class="block mb-1.5 text-gray-700 font-bold">Name</label>
              <n-input
                v-model:value="form.name"
                placeholder="e.g. local-mysql"
                class="w-full"
              />
            </div>

            <!-- Auth form tabs (dynamic per driver) -->
            <div v-if="Object.keys(authForms).length > 0">
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

            <!-- Fallback raw credential input (for plugins without structured auth forms) -->
            <div v-else>
              <label class="block mb-1.5 text-gray-700 font-bold">Connection String</label>
              <n-input
                v-model:value="rawCred"
                type="textarea"
                placeholder="DSN or connection string"
                :autosize="{ minRows: 3, maxRows: 8 }"
                class="w-full font-mono text-sm"
              />
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Bottom action bar, always visible -->
    <div class="shrink-0 p-4 bg-white border-t border-slate-200 shadow-sm">
      <n-flex justify="space-between" align="center">
        <n-button class="w-32" quaternary @click="CloseEditConnectionWindow">
          Cancel
        </n-button>
        <n-flex align="center" class="gap-3">
          <n-button
            class="w-40"
            :disabled="!canSave || testingConnection"
            :loading="testingConnection"
            @click="testConnection"
          >
            Test Connection
          </n-button>
          <n-button
            class="w-32"
            type="primary"
            :disabled="!canSave || saving"
            :loading="saving"
            @click="saveConnection"
          >
            Save
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
