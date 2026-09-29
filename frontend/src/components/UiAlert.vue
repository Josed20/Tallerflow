<script setup lang="ts">
defineProps<{ message: string }>()
</script>

<template>
  <p class="ui-alert" role="alert" aria-live="assertive">{{ message }}</p>
</template>
