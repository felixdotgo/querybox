<script setup>
import { Flash, Grid, Layers, Server, Terminal } from '@/lib/icons'
</script>

<template>
  <div class="h-full overflow-auto flex items-start justify-center bg-white p-8 pt-14">
    <div class="w-full max-w-xl">
      <!-- Header -->
      <div class="mb-10 text-center">
        <div class="inline-flex items-center justify-center w-14 h-14 rounded-2xl bg-blue-600 mb-5 shadow">
          <n-icon :size="30" color="white">
            <Terminal />
          </n-icon>
        </div>
        <h1 class="text-2xl font-bold text-gray-800 tracking-tight">
          QueryBox
        </h1>
      </div>

      <!-- Get started -->
      <div class="mb-8">
        <p class="text-[11px] font-semibold uppercase tracking-widest text-gray-400 mb-3">
          Get started
        </p>
        <div class="rounded-xl border border-gray-100 overflow-hidden divide-y divide-gray-100 shadow-sm">
          <div class="flex items-start gap-3.5 px-4 py-3.5 bg-gray-50">
            <n-icon :size="18" class="mt-0.5 text-blue-500 flex-shrink-0">
              <Server />
            </n-icon>
            <div>
              <p class="text-sm font-medium text-gray-700">
                Add a connection
              </p>
              <p class="text-xs text-gray-400 mt-0.5">
                Click <span class="font-semibold text-gray-500">+</span> in the Connections panel on the left to configure a new database connection.
              </p>
            </div>
          </div>
          <div class="flex items-start gap-3.5 px-4 py-3.5">
            <n-icon :size="18" class="mt-0.5 text-indigo-500 flex-shrink-0">
              <Layers />
            </n-icon>
            <div>
              <p class="text-sm font-medium text-gray-700">
                Browse your schema
              </p>
              <p class="text-xs text-gray-400 mt-0.5">
                Expand the connection tree to explore databases, schemas, tables and columns.
              </p>
            </div>
          </div>
          <div class="flex items-start gap-3.5 px-4 py-3.5 bg-gray-50">
            <n-icon :size="18" class="mt-0.5 text-green-500 flex-shrink-0">
              <Flash />
            </n-icon>
            <div>
              <p class="text-sm font-medium text-gray-700">
                Run a query
              </p>
              <p class="text-xs text-gray-400 mt-0.5">
                Click a table action or open a query tab. Press
                <kbd class="inline-flex items-center px-1.5 py-0.5 rounded bg-gray-100 border border-gray-200 text-[10px] font-mono text-gray-600 mx-0.5">Ctrl</kbd>+<kbd class="inline-flex items-center px-1.5 py-0.5 rounded bg-gray-100 border border-gray-200 text-[10px] font-mono text-gray-600 mx-0.5">Enter</kbd>
                to execute.
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Supported backends -->
      <div>
        <p class="text-[11px] font-semibold uppercase tracking-widest text-gray-400 mb-3">
          Supported databases
        </p>
        <div class="flex flex-wrap gap-2">
          <n-tag
            v-for="db in ['PostgreSQL', 'MySQL', 'SQLite', 'Redis', 'ArangoDB']"
            :key="db"
            size="small"
            :bordered="false"
            class="!bg-gray-100 !text-gray-600"
          >
            <template #icon>
              <n-icon><Grid /></n-icon>
            </template>
            {{ db }}
          </n-tag>
        </div>
      </div>
    </div>
  </div>
</template>
