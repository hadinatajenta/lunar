<template>
  <main class="page">
    <div class="ambient" aria-hidden="true" />
    <AuthHero />
    <section class="panel" aria-label="Create account">
      <RegisterForm />
    </section>
  </main>
</template>

<script setup lang="ts">
import AuthHero from "../components/AuthHero.vue"
import RegisterForm from "../components/RegisterForm.vue"
</script>

<style scoped>
.page {
  position: relative;
  min-height: 100vh;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(440px, 0.68fr);
  isolation: isolate;
}

.panel {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 0;
  padding: 40px;
  border-left: 1px solid var(--border);
  background: var(--surface);
  backdrop-filter: blur(14px);
}

@media (max-width: 980px) {
  .page {
    grid-template-columns: 1fr;
  }

  .panel {
    min-height: calc(100vh - 390px);
    padding: 40px 24px 56px;
    border-top: 1px solid var(--border);
    border-left: 0;
  }
}

@media (max-width: 520px) {
  .panel {
    min-height: auto;
    padding: 36px 22px 48px;
  }
}
</style>
