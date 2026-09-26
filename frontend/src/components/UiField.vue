<script setup lang="ts">
defineProps<{
  id: string
  label: string
  type?: string
  modelValue: string
  autocomplete?: string
  error?: string
  placeholder?: string
}>()

defineEmits<{ 'update:modelValue': [value: string] }>()
</script>

<template>
  <div class="ui-field">
    <label :for="id">{{ label }}</label>
    <input
      :id="id"
      :value="modelValue"
      :type="type ?? 'text'"
      :autocomplete="autocomplete"
      :placeholder="placeholder"
      :aria-invalid="Boolean(error)"
      :aria-describedby="error ? `${id}-error` : undefined"
      @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    />
    <p v-if="error" :id="`${id}-error`" class="ui-field__error" role="alert">{{ error }}</p>
  </div>
</template>
