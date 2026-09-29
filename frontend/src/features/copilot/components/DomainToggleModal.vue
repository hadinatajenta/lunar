<script setup lang="ts">
import { useCopilot } from "../composables/useCopilot"

const { defaultDomains, isDomainModalOpen, toggleDomain, isDomainEnabled } = useCopilot()
</script>

<template>
  <div v-if="isDomainModalOpen" class="modal-backdrop" @click.self="isDomainModalOpen = false">
    <div class="modal-card">
      <div class="modal-header">
        <div>
          <div class="modal-title">Active Copilot Tools</div>
          <div class="modal-sub">Select which BRI systems Copilot can query during reasoning.</div>
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
          @click="toggleDomain(domain.id)"
        >
          <div class="domain-info">
            <div class="domain-name">{{ domain.name }}</div>
            <div class="domain-desc">{{ domain.description }}</div>
          </div>

          <div class="toggle-switch" :class="{ 'is-on': isDomainEnabled(domain.id) }">
            <span class="toggle-handle"></span>
          </div>
        </div>
      </div>

      <div class="modal-footer">
        <span class="footer-note">Changes persist automatically in your browser profile.</span>
        <button class="done-btn" type="button" @click="isDomainModalOpen = false">
          Done
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  z-index: 50;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: rgba(0, 0, 0, 0.72);
  backdrop-filter: blur(8px);
}

.modal-card {
  width: min(100%, 520px);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius);
  background: #11141a;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.6);
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border);
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

.close-btn {
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--subtle);
  cursor: pointer;
  transition: color 180ms ease;
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
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 14px;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: rgba(255, 255, 255, 0.015);
  cursor: pointer;
  transition: background-color 180ms ease, border-color 180ms ease;
}

.domain-row:hover {
  background: rgba(255, 255, 255, 0.035);
  border-color: var(--border-strong);
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

.done-btn {
  height: 34px;
  padding: 0 16px;
  border: 0;
  border-radius: 7px;
  background: var(--accent);
  color: #0b0c0f;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 180ms ease;
}

.done-btn:hover {
  opacity: 0.9;
}
</style>
