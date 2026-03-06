<script setup>
import { computed } from 'vue'

const { schema } = defineProps({
  schema: {
    type: Object,
    required: true,
  },
})

const columnColumns = computed(() => [
  { title: 'Name', key: 'name', align: 'left' },
  { title: 'Type', key: 'type', align: 'left' },
  { title: 'Nullable', key: 'nullable', align: 'center', render: row => row.nullable ? 'YES' : 'NO' },
  { title: 'PK', key: 'primaryKey', align: 'center', render: row => row.primaryKey ? '✓' : '' },
  { title: 'Default', key: 'default', align: 'left' },
])

const indexColumns = computed(() => [
  { title: 'Name', key: 'name', align: 'left' },
  { title: 'Columns', key: 'columns', align: 'left', render: row => (row.columns || []).join(', ') },
  { title: 'Unique', key: 'unique', align: 'center', render: row => row.unique ? '✓' : '' },
  { title: 'Primary', key: 'primary', align: 'center', render: row => row.primary ? '✓' : '' },
])
</script>

<template>
  <div class="flex flex-col h-full w-full overflow-auto p-4">
    <h3 class="text-lg font-medium mb-4">
      {{ schema.name }}
    </h3>
    <div class="mb-6">
      <h4 class="font-semibold mb-2">
        Columns
      </h4>
      <n-data-table
        :columns="columnColumns"
        :data="schema.columns || []"
        size="small"
        bordered
        striped
        resizable
        class="w-full"
      />
    </div>
    <div>
      <h4 class="font-semibold mb-2">
        Indexes
      </h4>
      <n-data-table
        :columns="indexColumns"
        :data="schema.indexes || []"
        size="small"
        bordered
        striped
        resizable
        class="w-full"
      />
    </div>
  </div>
</template>

<style scoped>
/* nothing special yet */
</style>
