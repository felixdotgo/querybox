<template>
  <div class="container-fluid p-0 flex flex-col h-screen bg-white">
    <!-- Header (application menu placeholder) -->
    <header
      ref="headerRef"
      :class="['w-full border-b border-gray-200 pr-2 flex items-center flex-shrink-0', isMac ? 'pl-30 py-4 gap-2' : 'pl-2 gap-4 py-2']"
    >
        <n-button size="tiny" quaternary>File</n-button>
        <n-button size="tiny" quaternary>Edit</n-button>
        <n-button size="tiny" quaternary>View</n-button>
        <n-button size="tiny" quaternary>Help</n-button>
    </header>

    <!-- Content: two-column resizable layout -->
    <main class="flex-1 min-h-0">
      <div
        ref="containerRef"
        class="flex w-full h-full overflow-hidden"
      >
        <!-- Left column: toolbar + tree -->
        <div
          class="flex-shrink-0"
          :style="{ width: leftWidth + 'px', minWidth: '400px' }"
        >
          <div class="p-3 h-full flex flex-col gap-3">
            <!-- small toolbar -->
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-2">
                <h3 class="text-lg font-semibold m-0">Connections</h3>
                <small class="opacity-70">Tree</small>
              </div>
              <div class="flex items-center gap-2">
                <n-button
                  size="small"
                  secondary
                  title="New connection"
                  @click="openConnections"
                  >+</n-button
                >
              </div>
            </div>

            <n-input
              v-model:value="filter"
              size="small"
              placeholder="Filter connections"
            />

            <div class="flex-1 overflow-auto mt-2 px-1 min-h-0">
              <n-tree
                :data="filteredTreeData"
                :default-expanded-keys="defaultExpandedKeys"
                node-key="key"
                block-node
                :show-selector="false"
                @select="handleSelect"
              />
              <div
                v-if="connections.length === 0"
                class="py-6 text-center opacity-70"
              >
                No connections configured
              </div>
            </div>
          </div>
        </div>

        <!-- Drag divider -->
        <div
          role="separator"
          aria-orientation="vertical"
          class="w-1 cursor-col-resize bg-gray-200 hover:bg-sky-500"
          @pointerdown="startDrag"
        >
          <div class="w-0.5 h-full bg-transparent mx-auto"></div>
        </div>

        <!-- Right column: placeholder (functional details to be decided later) -->
        <div class="flex-1 p-6 min-h-0 overflow-auto">
          <h3 class="text-lg font-semibold mb-2">Workspace</h3>
          <div
            class="p-6 border border-dashed border-gray-200 rounded h-full flex items-center justify-center text-gray-500"
          >
            <div class="text-center">
              <div class="mb-2">
                Right pane placeholder — functionality to be decided
              </div>
              <div v-if="selectedConnection" class="mt-4 text-left text-sm">
                <div><strong>Name:</strong> {{ selectedConnection.name }}</div>
                <div>
                  <strong>Driver:</strong> {{ selectedConnection.driver_type }}
                </div>
                <div class="opacity-70 text-xs mt-2">
                  (Selecting a connection will later open editors / query tabs)
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>

    <!-- Footer resizer -->
    <div
      class="h-1 -mt-1 cursor-row-resize bg-gray-200 hover:bg-sky-500"
      @pointerdown="startFooterDrag"
    >
      <div class="w-10 h-0.5 bg-transparent mx-auto"></div>
    </div>

    <!-- Footer: collapsible log / debug area -->
    <footer class="w-full bg-white transition-all duration-200">
      <div class="flex items-center justify-between px-4 py-2">
        <div class="flex items-center gap-3">
          <button class="text-sm text-gray-600" @click="toggleFooter">
            <span v-if="footerCollapsed">Show logs</span>
            <span v-else>Hide logs</span>
          </button>
          <div class="text-xs opacity-60">Console / logs</div>
        </div>
        <div class="text-xs opacity-60">Status: ready</div>
      </div>

      <div
        v-show="!footerCollapsed"
        class="bg-gray-50 p-3 overflow-auto"
        :style="{ height: footerHeight + 'px' }"
      >
        <div class="text-sm text-gray-600">Log viewer (placeholder)</div>
        <div class="mt-2 text-xs opacity-60">
          Application logs, debug output and runtime diagnostics will appear
          here.
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from "vue"
import { useRouter } from "vue-router"
import { ListConnections } from "@/bindings/github.com/felixdotgo/querybox/services/connectionservice"
import { ShowConnectionsWindow, MinimiseMainWindow, ToggleFullScreenMainWindow, CloseMainWindow } from "@/bindings/github.com/felixdotgo/querybox/services/app"
import { createHorizontalResizer, createVerticalResizer } from "@/composables/useResize"

const isMac = navigator.userAgent.includes('Mac')

const router = useRouter()
async function openConnections() {
  try {
    await ShowConnectionsWindow()
    return
  } catch (err) {
    router.push("/connections")
  }
}

const headerRef = ref(null)
const containerRef = ref(null)
const leftWidth = ref(0)
const connections = ref([])
const filter = ref("")
const selectedConnection = ref(null)
const footerCollapsed = ref(true)
const footerHeight = ref(176)

// reusable resizers (horizontal + vertical)
const horizontalResizer = createHorizontalResizer({
  containerRef,
  sizeRef: leftWidth,
  min: 400,
  minOther: 200,
})

const verticalResizer = createVerticalResizer({
  sizeRef: footerHeight,
  min: 80,
  getMax: () => {
    const headerH = headerRef.value?.getBoundingClientRect().height ?? 0
    const minMainHeight = 120
    return Math.max(80, window.innerHeight - headerH - minMainHeight)
  },
})

const defaultExpandedKeys = computed(() => {
  return treeData.value.map((g) => g.key)
})

const treeData = computed(() => {
  const groups = {}
  for (const c of connections.value || []) {
    const key = c.driver_type || "unknown"
    if (!groups[key]) groups[key] = []
    groups[key].push(c)
  }
  return Object.entries(groups).map(([driver, conns]) => ({
    key: `driver:${driver}`,
    label: `${driver} (${conns.length})`,
    children: conns.map((cc) => ({ key: cc.id, label: cc.name })),
  }))
})

const filteredTreeData = computed(() => {
  const q = (filter.value || "").toLowerCase().trim()
  if (!q) return treeData.value
  return treeData.value
    .map((g) => ({
      ...g,
      children: g.children.filter((ch) =>
        (ch.label || "").toLowerCase().includes(q),
      ),
    }))
    .filter(
      (g) => g.children.length > 0 || (g.label || "").toLowerCase().includes(q),
    )
})

async function loadConnections() {
  try {
    connections.value = (await ListConnections()) || []
  } catch (err) {
    console.error("ListConnections", err)
    connections.value = []
  }
}

function handleSelect(key, node) {
  // leaf keys are connection IDs
  const conn = connections.value.find((c) => c.id === key)
  selectedConnection.value = conn || null
}

function startDrag(e) {
  horizontalResizer.start(e)
}

function startFooterDrag(e) {
  if (footerCollapsed.value) footerCollapsed.value = false
  verticalResizer.start(e)
}

// (removed) onPointerMove — using dedicated listeners for horizontal and vertical drags

// resizing cleanup handled by composables

function toggleFooter() {
  footerCollapsed.value = !footerCollapsed.value
  if (
    !footerCollapsed.value &&
    (!footerHeight.value || footerHeight.value < 80)
  ) {
    footerHeight.value = 176
  }
}

const resizeHandler = () => {
  horizontalResizer.clamp()
  verticalResizer.clamp()
}

onMounted(async () => {
  // initialize left column to 1/5 of container but at least 400px
  const rect = containerRef.value?.getBoundingClientRect()
  const cw = rect?.width ?? window.innerWidth
  leftWidth.value = Math.max(Math.floor(cw * 0.2), 400)

  // default footer height
  footerHeight.value = 176

  await loadConnections()

  // ensure initial sizes are within computed bounds
  horizontalResizer.clamp()
  verticalResizer.clamp()

  window.addEventListener("resize", resizeHandler)
})

onUnmounted(() => {
  window.removeEventListener("resize", resizeHandler)
  horizontalResizer.destroy()
  verticalResizer.destroy()
})
</script>
