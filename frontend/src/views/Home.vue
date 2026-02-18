<template>
  <div>
    <h3>Welcome to QueryBox</h3>
    <p class="opacity-80">Manage database connections and run ad-hoc queries from the Connections window.</p>
    <div class="mt-3">
      <n-button type="primary" @click="openConnections">Open Connections</n-button>
    </div>
  </div>
</template>

<script setup>
import { ShowConnections } from "@/bindings/github.com/felixdotgo/querybox/services/app";
import { useRouter } from "vue-router";

const router = useRouter();
async function openConnections() {
  try {
    await ShowConnections();
    return;
  } catch (err) {
    // binding not available (web) — navigate
    router.push('/connections');
  }
}
</script>
