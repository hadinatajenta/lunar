<template>
  <div class="login-card">
    <header class="login-header">
      <h2>Welcome back</h2>
      <p>Sign in to continue to your Lunar workspace.</p>
    </header>

    <div v-if="errorMessage" class="error-banner" role="alert">
      <span>{{ errorMessage }}</span>
    </div>

    <form @submit.prevent="handleSubmit" novalidate>
      <Input
        id="email"
        v-model="email"
        type="email"
        label="Email"
        placeholder="you@example.com"
        autocomplete="email"
        required
        :error="emailError"
        :disabled="isLoading"
      />

      <Input
        id="password"
        v-model="password"
        type="password"
        label="Password"
        placeholder="Enter your password"
        autocomplete="current-password"
        required
        :error="passwordError"
        :disabled="isLoading"
      />

      <div class="form-meta">
        <Checkbox
          id="remember"
          v-model="rememberMe"
          label="Remember me"
          :disabled="isLoading"
        />

        <router-link to="/forgot-password" class="link">
          Forgot password?
        </router-link>
      </div>

      <Button
        type="submit"
        variant="primary"
        :loading="isLoading"
        aria-label="Sign in"
      >
        Sign in
      </Button>
    </form>

    <div class="account-footer">
      Need an account?
      <router-link to="/register" class="access-link">
        Create one
      </router-link>
    </div>

    <p class="legal">
      By continuing, you agree to Lunar's terms and privacy policy.
    </p>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue"
import { useRouter } from "vue-router"
import Button from "@/components/ui/Button.vue"
import Checkbox from "@/components/ui/Checkbox.vue"
import Input from "@/components/ui/Input.vue"
import { useAuth } from "../composables/useAuth"

const router = useRouter()
const { login, isLoading, errorMessage, clearError } = useAuth()

const email = ref("")
const password = ref("")
const rememberMe = ref(false)

const emailError = ref<string | undefined>(undefined)
const passwordError = ref<string | undefined>(undefined)

const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

function validateForm(): boolean {
  clearError()
  emailError.value = undefined
  passwordError.value = undefined

  let isValid = true

  const trimmedEmail = email.value.trim()
  if (!trimmedEmail) {
    emailError.value = "Email is required"
    isValid = false
  } else if (!EMAIL_REGEX.test(trimmedEmail)) {
    emailError.value = "Please enter a valid email address"
    isValid = false
  }

  if (!password.value) {
    passwordError.value = "Password is required"
    isValid = false
  } else if (password.value.length < 6) {
    passwordError.value = "Password must be at least 6 characters"
    isValid = false
  }

  return isValid
}

async function handleSubmit(): Promise<void> {
  if (!validateForm()) {
    return
  }

  const success = await login({
    email: email.value.trim(),
    password: password.value,
    remember: rememberMe.value
  })

  if (success) {
    router.push("/dashboard")
  }
}
</script>

<style scoped>
.login-card {
  width: min(100%, 400px);
}

.login-header {
  margin-bottom: 30px;
}

.login-header h2 {
  color: var(--text);
  margin: 0;
  font-size: 28px;
  font-weight: 600;
  line-height: 1.2;
  letter-spacing: -0.04em;
}

.login-header p {
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

.form-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  margin: 4px 0 22px;
}

.link {
  color: var(--muted);
  font-size: 13px;
  text-decoration: none;
  transition: color 180ms ease;
}

.link:hover {
  color: var(--text);
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
  .login-header h2 {
    font-size: 26px;
  }
}
</style>
