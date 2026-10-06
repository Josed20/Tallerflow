<script setup lang="ts">
import { computed } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import AppShell from './modules/session/AppShell.vue'

const route = useRoute()
const usesInternalShell = computed(() => route.path.startsWith('/app'))
</script>

<template>
  <RouterView v-slot="{ Component }">
    <AppShell v-if="usesInternalShell">
      <component :is="Component" />
    </AppShell>
    <component :is="Component" v-else />
  </RouterView>
</template>
