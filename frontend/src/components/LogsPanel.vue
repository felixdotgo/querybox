<template>
  <div class="flex flex-col h-full font-mono text-xs select-text">
    <!-- Toolbar -->
    <div class="flex items-center gap-2 px-3 py-1.5 border-b border-gray-200 bg-gray-50 flex-shrink-0">
      <!-- Title + count -->
      <span class="font-semibold text-gray-500 uppercase tracking-widest text-[10px]">Logs</span>
      <n-text depth="3" class="text-[10px]">{{ filteredLogs.length }} entries</n-text>

      <div class="flex-1" />


      <!-- Auto-scroll toggle -->
      <n-button
        size="tiny"
        :type="autoScroll ? 'primary' : 'default'"
        tertiary
        title="Toggle auto-scroll"
        @click="autoScroll = !autoScroll"
      >
        <template #icon>
          <n-icon><ArrowDownOutline /></n-icon>
        </template>
        Auto-scroll
      </n-button>

      <n-divider vertical />

      <!-- Clear -->
      <n-button
        size="tiny"
        type="default"
        tertiary
        title="Clear logs"
        @click="$emit('clear')"
      >
        <template #icon>
          <n-icon><TrashOutline /></n-icon>
        </template>
        Clear
      </n-button>
    </div>

    <!-- Log list -->
    <n-scrollbar
      ref="scrollRef"
      class="flex-1"
      @scroll="onScroll"
    >
      <div v-if="filteredLogs.length === 0" class="flex items-center justify-center py-8">
        <n-empty description="No log entries" size="small" />
      </div>
      <table v-else class="w-full border-collapse">
        <tbody>
          <tr
            v-for="(entry, i) in filteredLogs"
            :key="i"
            class="border-b border-gray-100 hover:bg-gray-50 transition-colors"
          >
            <!-- Timestamp -->
            <td class="pl-3 pr-2 py-0.5 text-gray-400 whitespace-nowrap align-top w-32">
              {{ formatTime(entry.timestamp) }}
            </td>

            <!-- Level badge -->
            <td class="pr-3 py-0.5 align-top w-16">
              <n-tag
                :type="levelTagType(entry.level)"
                size="small"
                :bordered="false"
                style="font-size:10px;padding:0 5px;font-weight:700;text-transform:uppercase;"
              >{{ entry.level ?? 'info' }}</n-tag>
            </td>

            <!-- Source (optional) -->
            <td v-if="entry.source" class="pr-3 py-0.5 whitespace-nowrap align-top max-w-32 truncate">
              <n-text depth="3" class="text-[10px]">{{ entry.source }}</n-text>
            </td>

            <!-- Message -->
            <td class="pr-3 py-0.5 align-top break-all leading-5">
              <n-text>{{ entry.message }}</n-text>
            </td>
          </tr>
        </tbody>
      </table>
      <div ref="bottomRef" />
    </n-scrollbar>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick } from "vue"
import { ArrowDownOutline, TrashOutline } from "@vicons/ionicons5"

const props = defineProps({
  logs: { type: Array, default: () => [] },
})

defineEmits(["clear"])

const allLevels = ["debug", "info", "warn", "error"]
const activeLevels = ref(["debug", "info", "warn", "error"])
const autoScroll = ref(true)

const scrollRef = ref(null)
const bottomRef = ref(null)

const filteredLogs = computed(() =>
  props.logs.filter((e) => activeLevels.value.includes((e.level ?? "info").toLowerCase()))
)

function toggleLevel(lvl) {
  const idx = activeLevels.value.indexOf(lvl)
  if (idx === -1) {
    activeLevels.value.push(lvl)
  } else if (activeLevels.value.length > 1) {
    // keep at least one level active
    activeLevels.value.splice(idx, 1)
  }
}

function onScroll(e) {
  const el = e?.target
  if (!el) return
  const { scrollTop, scrollHeight, clientHeight } = el
  autoScroll.value = scrollHeight - scrollTop - clientHeight < 40
}

watch(
  () => filteredLogs.value.length,
  () => {
    if (autoScroll.value) {
      nextTick(() => bottomRef.value?.scrollIntoView({ behavior: "smooth" }))
    }
  }
)

// Format an RFC3339/RFC3339Nano timestamp to HH:MM:SS.mmm
function formatTime(ts) {
  if (!ts) return ""
  try {
    const normalized = ts.replace(/(\.\d{3})\d+/, "$1")
    const d = new Date(normalized)
    if (isNaN(d.getTime())) return ts
    const hh = String(d.getHours()).padStart(2, "0")
    const mm = String(d.getMinutes()).padStart(2, "0")
    const ss = String(d.getSeconds()).padStart(2, "0")
    const ms = String(d.getMilliseconds()).padStart(3, "0")
    return `${hh}:${mm}:${ss}.${ms}`
  } catch {
    return ts
  }
}

// Maps a log level to a NaiveUI tag type
function levelTagType(level) {
  switch ((level ?? "info").toLowerCase()) {
    case "error": return "error"
    case "warn":  return "warning"
    case "debug": return "info"
    default:      return "success"
  }
}
</script>
