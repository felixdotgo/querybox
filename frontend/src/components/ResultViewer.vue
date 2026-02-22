<template>
  <div class="w-full overflow-auto">
    <template v-if="result?.columns">
      <n-data-table
        :columns="tableColumns"
        :data="tableData"
        size="small"
        bordered
        striped
        scrollable
        class="w-full"
      />
    </template>
    <template v-else-if="result?.document">
      <pre class="whitespace-pre-wrap">{{ result.document | json }}</pre>
    </template>
    <template v-else-if="result?.kv">
      <n-descriptions bordered column="1">
        <n-descriptions-item
          v-for="(v, k) in result.kv.data || {}"
          :key="k"
          :label="k"
        >
          {{ v }}
        </n-descriptions-item>
      </n-descriptions>
    </template>
    <template v-else>
      <div class="text-gray-500">No result available</div>
    </template>
  </div>
</template>

<script setup>
import { computed } from "vue"

const props = defineProps({
  result: {
    type: Object,
    required: true,
  },
})

const result = props.result || {}

const tableColumns = computed(() => {
  const cols = result.columns || []
  return cols.map((c, idx) => ({
    title: c.name || `col${idx}`,
    key: `col${idx}`,
    align: "left",
    ellipsis: true,
  }))
})

const tableData = computed(() => {
  const rows = result.rows || []
  return rows.map((r, rowIdx) => {
    const obj = { key: rowIdx }
    (r.values || []).forEach((v, i) => {
      obj[`col${i}`] = v
    })
    return obj
  })
})
</script>
