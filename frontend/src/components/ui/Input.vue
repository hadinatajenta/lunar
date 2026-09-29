<template>
  <div class="field">
    <label v-if="label" :for="id" class="field-label">
      {{ label }}
    </label>
    <input
      :id="id"
      :name="name || id"
      :type="type"
      :value="modelValue"
      :placeholder="placeholder"
      :autocomplete="autocomplete"
      :required="required"
      :disabled="disabled"
      :aria-invalid="Boolean(error)"
      :aria-describedby="error ? `${id}-error` : undefined"
      :class="['field-input', { 'input-error': Boolean(error) }]"
      @input="handleInput"
    />
    <p v-if="error" :id="`${id}-error`" class="error-text" role="alert">
      {{ error }}
    </p>
  </div>
</template>

<script setup lang="ts">
interface InputProps {
  id: string
  name?: string
  label?: string
  modelValue?: string | number
  type?: string
  placeholder?: string
  autocomplete?: string
  required?: boolean
  disabled?: boolean
  error?: string
}

withDefaults(defineProps<InputProps>(), {
  name: undefined,
  label: undefined,
  modelValue: "",
  type: "text",
  placeholder: undefined,
  autocomplete: undefined,
  required: false,
  disabled: false,
  error: undefined
})

const emit = defineEmits<{
  (e: "update:modelValue", value: string): void
}>()

function handleInput(event: Event): void {
  const target = event.target as HTMLInputElement
  emit("update:modelValue", target.value)
}
</script>

<style scoped>
.field {
  margin-bottom: 18px;
}

.field-label {
  display: block;
  margin-bottom: 8px;
  color: #d8dde4;
  font-size: 13px;
  font-weight: 500;
}

.field-input {
  width: 100%;
  height: 48px;
  padding: 0 14px;
  border: 1px solid var(--border);
  border-radius: 10px;
  outline: none;
  background: var(--surface-raised);
  color: var(--text);
  font-size: 14px;
  font-family: inherit;
  transition:
    border-color 180ms ease,
    background-color 180ms ease,
    box-shadow 180ms ease;
}

.field-input::placeholder {
  color: #5d6571;
}

.field-input:hover:not(:disabled) {
  border-color: rgba(255, 255, 255, 0.11);
}

.field-input:focus {
  border-color: var(--border-strong);
  background: #181c24;
  box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.04);
}

.field-input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.input-error {
  border-color: #ef4444 !important;
}

.input-error:focus {
  box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.15) !important;
}

.error-text {
  margin: 6px 0 0;
  color: #f87171;
  font-size: 12px;
  line-height: 1.4;
}
</style>
