<script setup lang="ts">
defineProps<{
  id: string
  name?: string
  label: string
  type?: string
  modelValue: string
  autocomplete?: string
  required?: boolean
  spellcheck?: boolean
  error?: string
  placeholder?: string
  description?: string
}>()

defineEmits<{ 'update:modelValue': [value: string] }>()
</script>

<template>
  <div class="ui-field">
    <label :for="id">{{ label }}</label>
    <input
      :id="id"
      :name="name ?? id"
      :value="modelValue"
      :type="type ?? 'text'"
      :autocomplete="autocomplete"
      :required="required"
      :spellcheck="spellcheck"
      :placeholder="placeholder"
      :aria-invalid="Boolean(error)"
      :aria-describedby="[description ? `${id}-description` : '', error ? `${id}-error` : ''].filter(Boolean).join(' ') || undefined"
      @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    />
    <p v-if="description" :id="`${id}-description`" class="ui-field__description">{{ description }}</p>
    <p v-if="error" :id="`${id}-error`" class="ui-field__error" role="alert">{{ error }}</p>
  </div>
</template>
