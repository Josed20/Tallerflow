<script setup lang="ts">
import ProductionSettingsPanel from './ProductionSettingsPanel.vue'
</script>
<template><main class="settings-page"><header><h1>Configuración del taller</h1><p>Define el proceso que seguirán los nuevos pedidos. Las órdenes existentes conservan sus etapas.</p></header><ProductionSettingsPanel /></main></template>
<style scoped>.settings-page{max-width:960px;margin:auto;padding:clamp(1rem,3vw,2.5rem)}h1{font-family:var(--tf-font-body);font-size:1.75rem;letter-spacing:-.03em}header p{color:var(--tf-text-subtle);max-width:65ch}</style>
