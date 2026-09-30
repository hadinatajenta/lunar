<template>
  <div class="register-card">
    <header class="register-header">
      <h2>Create your workspace account</h2>
      <p>Verify your BRI developer credentials to get started.</p>
    </header>

    <div v-if="displayError" class="error-banner" role="alert">
      <span>{{ displayError }}</span>
    </div>

    <form @submit.prevent="handleSubmit" novalidate>
      <div v-if="!verifiedProfile" class="step-container">
        <Input
          id="jira-pat"
          v-model="pat"
          type="password"
          label="Jira Personal Access Token"
          placeholder="Enter your Jira PAT"
          autocomplete="off"
          required
          :error="patError"
          :disabled="isLoading"
        />
        <p class="field-hint">Generate a token in your Jira BRI profile settings.</p>

        <Button
          type="button"
          variant="primary"
          :loading="isLoading"
          aria-label="Verify Identity"
          class="action-button"
          @click="handleVerifyPat"
        >
          Verify Identity
        </Button>
      </div>

      <div v-else class="step-container">
        <div class="verified-badge">
          <div class="badge-avatar">{{ initials }}</div>
          <div class="badge-details">
            <div class="badge-top">
              <span class="badge-name">{{ verifiedProfile.display_name }}</span>
              <span class="badge-tag">
                <svg
                  class="badge-icon"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="3"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  aria-hidden="true"
                >
                  <polyline points="20 6 9 17 4 12" />
                </svg>
                Verified
              </span>
            </div>
            <div class="badge-meta">
              <span class="badge-email">{{ verifiedProfile.email }}</span>
              <span class="badge-username">{{ verifiedProfile.username }}</span>
            </div>
          </div>
          <button
            type="button"
            class="change-button"
            :disabled="isLoading"
            aria-label="Change token"
            @click="handleResetToken"
          >
            Change
          </button>
        </div>

        <Input
          id="password"
          v-model="password"
          type="password"
          label="Create Password"
          placeholder="At least 6 characters"
          autocomplete="new-password"
          required
          :error="passwordError"
          :disabled="isLoading"
        />

        <Input
          id="confirm-password"
          v-model="confirmPassword"
          type="password"
          label="Confirm Password"
          placeholder="Repeat your password"
          autocomplete="new-password"
          required
          :error="confirmPasswordError"
          :disabled="isLoading"
        />

        <Button
          type="submit"
          variant="primary"
          :loading="isLoading"
          aria-label="Create Workspace Account"
          class="action-button"
        >
          Create Workspace Account
        </Button>
      </div>
    </form>

    <div class="account-footer">
      Already have an account?
      <router-link to="/login" class="access-link">
        Sign in
      </router-link>
    </div>

    <p class="legal">
      By continuing, you agree to Lunar's terms and privacy policy.
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue"
import { useRouter } from "vue-router"
import Button from "@/components/ui/Button.vue"
import Input from "@/components/ui/Input.vue"
import { useAuth } from "../composables/useAuth"
import type { VerifiedJiraProfile } from "../types"

const router = useRouter()
const { verifyJiraPat, registerWithJira, isLoading, errorMessage, clearError } = useAuth()

const pat = ref("")
const password = ref("")
const confirmPassword = ref("")
const verifiedProfile = ref<VerifiedJiraProfile | null>(null)

const patError = ref<string | undefined>(undefined)
const passwordError = ref<string | undefined>(undefined)
const confirmPasswordError = ref<string | undefined>(undefined)
const validationError = ref<string | null>(null)

const displayError = computed(() => validationError.value || errorMessage.value)

const initials = computed(() => {
  if (!verifiedProfile.value?.display_name) {
    return "BRI"
  }
  const parts = verifiedProfile.value.display_name.trim().split(/\s+/)
  if (parts.length === 1) {
    return parts[0].slice(0, 2).toUpperCase()
  }
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
})

function handleResetToken(): void {
  verifiedProfile.value = null
  password.value = ""
  confirmPassword.value = ""
  passwordError.value = undefined
  confirmPasswordError.value = undefined
  validationError.value = null
  clearError()
}

async function handleVerifyPat(): Promise<void> {
  clearError()
  validationError.value = null
  patError.value = undefined

  const trimmedPat = pat.value.trim()
  if (!trimmedPat) {
    patError.value = "Personal Access Token is required"
    return
  }

  const profile = await verifyJiraPat(trimmedPat)
  if (profile) {
    verifiedProfile.value = profile
  }
}

async function handleSubmit(): Promise<void> {
  clearError()
  validationError.value = null
  passwordError.value = undefined
  confirmPasswordError.value = undefined

  if (!verifiedProfile.value) {
    await handleVerifyPat()
    return
  }

  let isValid = true

  if (!password.value) {
    passwordError.value = "Password is required"
    isValid = false
  } else if (password.value.length < 6) {
    passwordError.value = "Password must be at least 6 characters"
    isValid = false
  }

  if (!confirmPassword.value) {
    confirmPasswordError.value = "Please confirm your password"
    isValid = false
  } else if (password.value !== confirmPassword.value) {
    confirmPasswordError.value = "Passwords do not match"
    isValid = false
  }

  if (!isValid) {
    return
  }

  const success = await registerWithJira(pat.value.trim(), password.value)
  if (success) {
    router.push("/dashboard")
  }
}
</script>

<style scoped>
.register-card {
  width: min(100%, 400px);
}

.register-header {
  margin-bottom: 30px;
}

.register-header h2 {
  color: var(--text);
  margin: 0;
  font-size: 28px;
  font-weight: 600;
  line-height: 1.2;
  letter-spacing: -0.04em;
}

.register-header p {
  margin: 10px 0 0;
  color: var(--muted);
  font-size: 14px;
  line-height: 1.55;
}

.error-banner {
  display: flex;
  align-items: center;
  margin-bottom: 20px;
  padding: 12px 14px;
  border: 1px solid rgba(239, 68, 68, 0.25);
  border-radius: 10px;
  background: rgba(239, 68, 68, 0.1);
  color: #fca5a5;
  font-size: 13px;
  line-height: 1.4;
}

.step-container {
  display: flex;
  flex-direction: column;
}

.field-hint {
  margin: 6px 0 0;
  color: var(--muted);
  font-size: 12px;
  line-height: 1.45;
}

.action-button {
  margin-top: 4px;
}

.verified-badge {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 22px;
  padding: 14px;
  border: 1px solid var(--border-strong);
  border-radius: 12px;
  background: var(--surface-raised);
}

.badge-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.08);
  color: var(--text);
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.badge-details {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
  gap: 3px;
}

.badge-top {
  display: flex;
  align-items: center;
  gap: 8px;
}

.badge-name {
  overflow: hidden;
  color: var(--text);
  font-size: 14px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.badge-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 7px;
  border-radius: 9999px;
  background: rgba(34, 197, 94, 0.15);
  color: #4ade80;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.badge-icon {
  width: 11px;
  height: 11px;
}

.badge-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  overflow: hidden;
}

.badge-email {
  overflow: hidden;
  color: var(--muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.badge-username {
  padding: 1px 5px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.06);
  color: var(--subtle);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
}

.change-button {
  flex-shrink: 0;
  padding: 4px 8px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  text-decoration: underline;
  transition: color 180ms ease;
}

.change-button:hover:not(:disabled) {
  color: var(--text);
}

.change-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.account-footer {
  margin: 28px 0 0;
  text-align: center;
  color: var(--muted);
  font-size: 12px;
  line-height: 1.5;
  text-align: center;
}

.access-link {
  margin-left: 4px;
  color: var(--muted);
  text-decoration: none;
  transition: color 180ms ease;
}

.access-link:hover {
  color: var(--text);
}

.legal {
  margin-top: 30px;
  color: var(--subtle);
  font-size: 11px;
  line-height: 1.5;
  text-align: center;
}

@media (max-width: 520px) {
  .register-header h2 {
    font-size: 26px;
  }
}
</style>
