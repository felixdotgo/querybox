<template>
  <div class="container-fluid p-0">
    <n-layout has-sider>
      <n-layout-sider class="border-r border-gray-200">
        <div class="p-4">
          <h3 class="mb-2">Drivers</h3>
          <ul class="list-none p-0 m-0 flex flex-col gap-1.5">
            <li v-if="drivers.length === 0" class="opacity-70">
              No driver plugins found
            </li>
            <li v-for="p in filteredDrivers" :key="p.name">
              <n-button
                block
                :type="
                  selectedPlugin && selectedPlugin.name === p.name
                    ? 'primary'
                    : 'default'
                "
                @click="selectPlugin(p)"
                class="flex justify-between items-center text-left p-2"
              >
                <div>
                  <div class="font-semibold">
                    {{ p.name }}
                    <small class="opacity-70 ml-1.5">{{
                      p.version || ""
                    }}</small>
                  </div>
                </div>
              </n-button>
            </li>
          </ul>
        </div>
      </n-layout-sider>

      <n-layout-content>
        <div class="p-4">
          <h3 class="mb-2">Connect to database</h3>
          <div class="flex flex-col gap-2 max-w-3xl">
            <div>
              <label class="block mb-1.5 text-gray-700">Driver</label>
              <n-input
                v-model:value="form.driver"
                readonly
                :placeholder="
                  selectedPlugin
                    ? selectedPlugin.name
                    : 'Select a driver from the left'
                "
                class="w-full"
              />
            </div>

            <div>
              <label class="block mb-1.5 text-gray-700">Connection name</label>
              <n-input
                v-model:value="form.name"
                placeholder="e.g. local-mysql"
                class="w-full"
              />
            </div>

            <div>
              <label class="block mb-1.5 text-gray-700">DSN / Credential</label>
              <n-input
                v-model:value="form.cred"
                placeholder="DSN (user:pass@tcp(host:port)/dbname) or plugin-specific"
                class="w-full"
              />
            </div>

            <div class="flex items-center gap-2 mt-2">
              <n-button type="primary" @click="connect" :disabled="!canConnect"
                >Connect</n-button
              >
              <n-button @click="clearForm">Clear</n-button>
              <div class="ml-auto text-gray-600">{{ statusText }}</div>
            </div>
          </div>
        </div>
      </n-layout-content>
    </n-layout>

    <div class="border-t p-4 border-gray-200 w-full flex">
      <n-button class="w-32 ml-auto" @click="closeWindow">Close</n-button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from "vue"
import {
  ListConnections,
  CreateConnection,
  DeleteConnection,
} from "@/bindings/github.com/felixdotgo/querybox/services/connectionservice"
import { ListPlugins } from "@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager"
import { CloseConnections } from "@/bindings/github.com/felixdotgo/querybox/services/app"

const connections = ref([])
const plugins = ref([])
const pluginFilter = ref("")
const selectedPlugin = ref(null)
const statusText = ref("")

const form = ref({ name: "", driver: "", cred: "" })
const filterText = ref("")

const filtered = computed(() => {
  const f = (filterText.value || "").toLowerCase()
  return (connections.value || []).filter((c) =>
    c.name.toLowerCase().includes(f),
  )
})

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
    const [plist, clist] = await Promise.all([ListPlugins(), ListConnections()])
    plugins.value = plist || []
    connections.value = clist || []
  } catch (err) {
    console.error("load:", err)
    plugins.value = plugins.value || []
    connections.value = connections.value || []
  }
}

function selectPlugin(p) {
  selectedPlugin.value = p
  form.value.driver = p.name || ""
  if (!form.value.name) {
    form.value.name = `${p.name}-connection`
  }
}

function clearForm() {
  form.value = { name: "", driver: "", cred: "" }
  selectedPlugin.value = null
  statusText.value = ""
}

async function connect() {
  if (!canConnect.value) {
    alert("Please select a driver, provide a connection name and DSN")
    return
  }
  try {
    statusText.value = "Connecting..."
    await CreateConnection(
      form.value.name.trim(),
      form.value.driver.trim(),
      form.value.cred.trim(),
    )
    statusText.value = "Saved"
    // reset form but keep driver selected
    form.value = { name: "", driver: "", cred: "" }
    await load()
    setTimeout(() => {
      statusText.value = ""
    }, 1200)
  } catch (err) {
    console.error("connect:", err)
    statusText.value = "Failed to save"
    alert("Failed to create connection")
  }
}

async function remove(id) {
  if (!confirm("Delete this connection?")) return
  try {
    await DeleteConnection(id)
    await load()
  } catch (err) {
    console.error("delete:", err)
    alert("Failed to delete connection")
  }
}

function refresh() {
  load()
}

function closeWindow() {
  // Best-effort: close the app window (works in Wails / browser fallback)
  try {
    CloseConnections()
  } catch (e) {
    /* ignore */
  }
}

onMounted(async () => {
  await load()
})
</script>
