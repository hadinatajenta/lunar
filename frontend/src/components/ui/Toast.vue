<script setup lang="ts">
import { useToast } from "../../composables/useToast"

const { toastMessage, isToastVisible } = useToast()
</script>

<template>
  <div
    id="toast"
    class="toast"
    :class="{ 'is-visible': isToastVisible }"
    role="status"
    aria-live="polite"
    data-testid="global-toast"
  >
    {{ toastMessage }}
  </div>
</template>

<style scoped>
.toast {
  position: fixed;
  right: 22px;
  bottom: 22px;
  z-index: 50;
  display: none;
  max-width: 340px;
  padding: 11px 15px;
  border: 1px solid var(--border-strong);
  border-radius: 9px;
  background: #151820;
  color: #dfe4e8;
  box-shadow: 0 14px 40px rgba(0, 0, 0, 0.45);
  font-size: 11.5px;
  line-height: 1.5;
  animation: fadeIn 180ms ease forwards;
}

.toast.is-visible {
  display: block;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(6px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
