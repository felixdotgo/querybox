<script setup>
import { Flash, Layers, Server, Terminal } from '@/lib/icons'
</script>

<template>
  <div class="h-full flex flex-col items-center justify-center">
    <div class="w-full max-w-xs flex flex-col items-center gap-8">
      <!-- Header -->
      <div class="flex items-center gap-2">
        <div class="inline-flex items-center justify-center w-10 h-10 rounded-xl bg-[#18a058]">
          <n-icon :size="22" color="white">
            <Terminal />
          </n-icon>
        </div>
        <div>
          <h1 class="text-base font-semibold text-slate-800 tracking-tight">
            QueryBox
          </h1>
        </div>
      </div>

      <!-- Steps -->
      <div class="w-full flex flex-col gap-3">
        <div class="flex items-start gap-3">
          <n-icon :size="14" class="mt-0.5 flex-shrink-0 text-[#18a058]">
            <Server />
          </n-icon>
          <div>
            <p class="text-sm font-medium text-slate-700">
              Add a connection
            </p>
            <p class="text-sm text-slate-400 mt-0.5 leading-relaxed">
              Click <span class="font-semibold text-slate-500">+</span> in the Connections panel to configure a database.
            </p>
          </div>
        </div>
        <div class="flex items-start gap-3">
          <n-icon :size="14" class="mt-0.5 flex-shrink-0 text-[#18a058]">
            <Layers />
          </n-icon>
          <div>
            <p class="text-sm font-medium text-slate-700">
              Browse your schema
            </p>
            <p class="text-sm text-slate-400 mt-0.5 leading-relaxed">
              Expand the tree to explore databases, tables and columns.
            </p>
          </div>
        </div>
        <div class="flex items-start gap-3">
          <n-icon :size="14" class="mt-0.5 flex-shrink-0 text-[#18a058]">
            <Flash />
          </n-icon>
          <div>
            <p class="text-sm font-medium text-slate-700">
              Run a query
            </p>
            <p class="text-sm text-slate-400 mt-0.5 leading-relaxed">
              Open a query tab and press
              <kbd class="inline-flex items-center px-1 py-px rounded bg-slate-100 border border-slate-300 text-[10px] font-mono text-slate-500 mx-0.5">Ctrl</kbd>+<kbd class="inline-flex items-center px-1 py-px rounded bg-slate-100 border border-slate-300 text-[10px] font-mono text-slate-500 mx-0.5">Enter</kbd>
              to execute.
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
