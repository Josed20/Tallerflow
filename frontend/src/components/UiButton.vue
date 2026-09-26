<script setup lang="ts">
withDefaults(defineProps<{ type?: 'button' | 'submit'; loading?: boolean }>(), {
  type: 'button',
  loading: false,
})
</script>

<template>
  <button class="ui-button" :type="type" :disabled="loading">
    <span v-if="loading" class="ui-button__spinner" aria-hidden="true" />
    <slot />
  </button>
</template>
