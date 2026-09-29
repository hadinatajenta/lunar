<script setup lang="ts">
import AppSidebar from "./AppSidebar.vue"
import AppHeader from "./AppHeader.vue"
</script>

<template>
  <div class="shell">
    <AppSidebar />
    <div class="main-column">
      <AppHeader />
      <main class="main-content">
        <slot />
      </main>
    </div>
  </div>
</template>

<style scoped>
.shell {
  min-height: 100vh;
  display: grid;
  grid-template-columns: 228px minmax(0, 1fr);
  background: var(--bg);
}

.main-column {
  min-width: 0;
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.main-content {
  flex: 1;
  min-width: 0;
  background:
    radial-gradient(circle at 90% 8%, rgba(255, 255, 255, 0.028), transparent 28%),
    var(--bg);
}

@media (max-width: 980px) {
  .shell {
    grid-template-columns: 1fr;
  }
}
</style>
