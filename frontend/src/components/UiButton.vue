<script setup lang="ts">
withDefaults(defineProps<{
  type?: 'button' | 'submit'
  loading?: boolean
  variant?: 'primary' | 'secondary' | 'danger'
}>(), {
  type: 'button',
  loading: false,
  variant: 'primary',
})
</script>

<template>
  <button :class="['ui-button', `ui-button--${variant}`]" :type="type" :disabled="loading">
    <span v-if="loading" class="ui-button__spinner" aria-hidden="true" />
    <slot />
  </button>
</template>
