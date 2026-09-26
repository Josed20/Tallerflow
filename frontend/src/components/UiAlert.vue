<script setup lang="ts">
defineProps<{ message: string }>()
</script>

<template>
  <p class="ui-alert" role="alert">{{ message }}</p>
</template>
