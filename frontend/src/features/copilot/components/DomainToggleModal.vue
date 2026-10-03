<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useCopilot } from "../composables/useCopilot"

const { defaultDomains, isDomainModalOpen, toggleDomain, isDomainEnabled, enableAllDomains } = useCopilot()

const isConfirmingEnableAll = ref(false)

const areAllDomainsEnabled = computed(() => defaultDomains.every((domain) => isDomainEnabled(domain.id)))

const enabledDomainCount = computed(() => defaultDomains.filter((domain) => isDomainEnabled(domain.id)).length)

watch(isDomainModalOpen, () => {
  isConfirmingEnableAll.value = false
})

const confirmEnableAll = () => {
  enableAllDomains()
  isConfirmingEnableAll.value = false
}
</script>

<template>
  <div v-if="isDomainModalOpen" class="modal-backdrop" @click.self="isDomainModalOpen = false">
    <div class="modal-card">
      <div class="modal-header">
        <div>
          <div class="modal-title">Active Copilot Tools</div>
          <div class="modal-sub">Select which BRI systems Copilot can query during reasoning.</div>
          <div class="modal-wire">{{ enabledDomainCount }} of {{ defaultDomains.length }} domains sent to the local helper</div>
        </div>
        <button class="close-btn" type="button" @click="isDomainModalOpen = false" aria-label="Close">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </button>
      </div>

      <div class="domain-list">
        <div
          v-for="domain in defaultDomains"
          :key="domain.id"
          class="domain-row"
          :class="{ 'is-active': isDomainEnabled(domain.id) }"
          role="switch"
          tabindex="0"
          :aria-checked="isDomainEnabled(domain.id)"
          @click="toggleDomain(domain.id)"
          @keydown.enter.prevent="toggleDomain(domain.id)"
          @keydown.space.prevent="toggleDomain(domain.id)"
        >
          <div class="domain-info">
            <div class="domain-name">{{ domain.name }}</div>
            <div class="domain-desc">{{ domain.description }}</div>
            <div class="domain-wire">{{ domain.id }}</div>
          </div>

          <div class="toggle-switch" :class="{ 'is-on': isDomainEnabled(domain.id) }" aria-hidden="true">
            <span class="toggle-handle"></span>
          </div>
        </div>
      </div>

      <div class="modal-footer">
        <span class="footer-note">Enabled tools are sent as the domains field, which gates the helper tool schemas.</span>
        <div class="actions-row">
          <button class="ghost-btn" type="button" :disabled="areAllDomainsEnabled" @click="isConfirmingEnableAll = true">
            Enable all tools
          </button>
          <button class="done-btn" type="button" @click="isDomainModalOpen = false">
            Done
          </button>
        </div>
      </div>

      <div v-if="isConfirmingEnableAll" class="enable-all-confirm">
        <p class="confirm-text" role="alert">Enabling all tools sends every tool to the model on each message, which uses more tokens.</p>
        <div class="actions-row">
          <button class="confirm-btn" type="button" @click="confirmEnableAll">Enable all</button>
          <button class="ghost-btn" type="button" @click="isConfirmingEnableAll = false">Cancel</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-backdrop {
  position: fixed; inset: 0; z-index: 50; display: flex; align-items: center; justify-content: center;
  padding: 20px; background: rgba(0, 0, 0, 0.72); backdrop-filter: blur(8px);
}

.modal-card {
  width: 100%; max-width: 480px; border: 1px solid var(--border); border-radius: 14px;
  background: var(--surface); color: var(--text); box-shadow: 0 20px 40px rgba(0, 0, 0, 0.6); overflow: hidden;
}

.modal-header {
  display: flex; align-items: flex-start; justify-content: space-between;
  padding: 20px 24px; border-bottom: 1px solid var(--border);
}

.modal-title {
  color: var(--text);
  font-size: 16px;
  font-weight: 600;
  letter-spacing: -0.01em;
}

.modal-sub {
  margin-top: 4px;
  color: var(--muted);
  font-size: 13px;
}

.modal-wire {
  margin-top: 6px;
  color: var(--positive);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.close-btn {
  width: 28px; height: 28px; display: grid; place-items: center; border: 0; border-radius: 6px;
  background: transparent; color: var(--subtle); cursor: pointer; transition: color 180ms ease;
}

.close-btn:hover {
  color: var(--text);
}

.close-btn svg {
  width: 16px;
  height: 16px;
}

.domain-list {
  padding: 12px 16px;
  display: grid;
  gap: 8px;
}

.domain-row {
  display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 12px 14px;
  border: 1px solid var(--border); border-radius: 9px; background: var(--surface-raised);
  cursor: pointer; transition: background-color 180ms ease, border-color 180ms ease;
}

.domain-row:hover {
  background: rgba(255, 255, 255, 0.035);
  border-color: var(--border-strong);
}

.domain-row:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.domain-row.is-active {
  background: rgba(255, 255, 255, 0.04);
}

.domain-name {
  color: var(--text);
  font-size: 13px;
  font-weight: 600;
}

.domain-desc {
  margin-top: 2px;
  color: var(--subtle);
  font-size: 11px;
}

.domain-wire {
  margin-top: 4px;
  color: var(--muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 10px;
}

.toggle-switch {
  width: 38px;
  height: 22px;
  padding: 2px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.1);
  transition: background-color 180ms ease;
  flex-shrink: 0;
}

.toggle-switch.is-on {
  background: var(--positive);
}

.toggle-handle {
  display: block;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: #ffffff;
  transition: transform 180ms ease;
}

.toggle-switch.is-on .toggle-handle {
  transform: translateX(16px);
}

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
  border-top: 1px solid var(--border);
}

.footer-note {
  color: var(--subtle);
  font-size: 11px;
}

.actions-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.ghost-btn {
  padding: 9px 14px;
  border: 1px solid var(--border-strong);
  border-radius: 8px;
  background: transparent;
  color: var(--text);
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
}

.ghost-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.done-btn,
.confirm-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 36px;
  padding: 0 16px;
  border: 0;
  border-radius: 8px;
  background: var(--accent);
  color: var(--accent-contrast);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 180ms ease;
}

.done-btn:hover,
.confirm-btn:hover {
  opacity: 0.9;
}

.enable-all-confirm {
  display: grid;
  gap: 10px;
  padding: 14px 24px 16px;
  border-top: 1px solid var(--border);
  background: var(--surface-raised);
}

.confirm-text {
  margin: 0;
  font-size: 12px;
  line-height: 1.5;
}
</style>
