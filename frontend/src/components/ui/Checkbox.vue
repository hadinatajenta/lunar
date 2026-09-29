<template>
  <label :for="id" :class="['checkbox-wrapper', { 'checkbox-disabled': disabled }]">
    <input
      :id="id"
      :name="name"
      type="checkbox"
      :checked="modelValue"
      :disabled="disabled"
      class="checkbox-input"
      @change="handleChange"
    />
    <span v-if="label" class="checkbox-label">{{ label }}</span>
    <slot v-else />
  </label>
</template>

<script setup lang="ts">
interface CheckboxProps {
  id?: string
  name?: string
  label?: string
  modelValue?: boolean
  disabled?: boolean
}

withDefaults(defineProps<CheckboxProps>(), {
  id: undefined,
  name: undefined,
  label: undefined,
  modelValue: false,
  disabled: false
})

const emit = defineEmits<{
  (e: "update:modelValue", value: boolean): void
}>()

function handleChange(event: Event): void {
  const target = event.target as HTMLInputElement
  emit("update:modelValue", target.checked)
}
</script>

<style scoped>
.checkbox-wrapper {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--muted);
  font-size: 13px;
  cursor: pointer;
  user-select: none;
}

.checkbox-input {
  width: 14px;
  height: 14px;
  margin: 0;
  accent-color: var(--accent);
  cursor: pointer;
}

.checkbox-disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.checkbox-disabled .checkbox-input {
  cursor: not-allowed;
}

.checkbox-label {
  line-height: 1.4;
}
</style>
