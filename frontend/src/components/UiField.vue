<script setup lang="ts">
import { Eye, EyeOff } from '@lucide/vue'
import { ref } from 'vue'

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
  togglePassword?: boolean
}>()

defineEmits<{ 'update:modelValue': [value: string] }>()
const passwordVisible = ref(false)
</script>

<template>
  <div class="ui-field">
    <label :for="id">{{ label }}</label>
    <div class="ui-field__control" :class="{ 'ui-field__control--password': togglePassword }">
      <input
        :id="id"
        :name="name ?? id"
        :value="modelValue"
        :type="togglePassword && passwordVisible ? 'text' : (type ?? 'text')"
        :autocomplete="autocomplete"
        :required="required"
        :spellcheck="spellcheck ?? (togglePassword ? false : undefined)"
        :placeholder="placeholder"
        :aria-invalid="Boolean(error)"
        :aria-describedby="[description ? `${id}-description` : '', error ? `${id}-error` : ''].filter(Boolean).join(' ') || undefined"
        @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
      />
      <button
        v-if="togglePassword"
        type="button"
        class="ui-field__visibility"
        :aria-label="passwordVisible ? 'Ocultar contraseña' : 'Mostrar contraseña'"
        :aria-controls="id"
        @click="passwordVisible = !passwordVisible"
      >
        <EyeOff v-if="passwordVisible" :size="20" aria-hidden="true" />
        <Eye v-else :size="20" aria-hidden="true" />
      </button>
    </div>
    <p v-if="description" :id="`${id}-description`" class="ui-field__description">{{ description }}</p>
    <p v-if="error" :id="`${id}-error`" class="ui-field__error" role="alert">{{ error }}</p>
  </div>
</template>

<style scoped>
.ui-field__control {
  position: relative;
}

.ui-field__control--password input {
  padding-inline-end: 3rem;
}

.ui-field__visibility {
  position: absolute;
  inset-block: 0;
  inset-inline-end: 0;
  display: grid;
  width: 44px;
  place-items: center;
  border: 0;
  border-radius: 0.5rem;
  background: transparent;
  color: var(--tf-text-muted);
  cursor: pointer;
}

.ui-field__visibility:focus-visible {
  outline: 3px solid var(--tf-focus);
  outline-offset: -3px;
}
</style>
