<script setup lang="ts">
defineProps<{
  name: string
}>()
</script>

<template>
  <svg v-if="name === 'dashboard'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
    <rect x="4" y="4" width="6" height="6" rx="1"></rect>
    <rect x="14" y="4" width="6" height="6" rx="1"></rect>
    <rect x="4" y="14" width="6" height="6" rx="1"></rect>
    <rect x="14" y="14" width="6" height="6" rx="1"></rect>
  </svg>
  <svg v-else-if="name === 'jira'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
    <path d="M7 4h10l-3 4H7z"></path>
    <path d="M7 20h10l-3-4H7z"></path>
    <path d="M17 4v6a4 4 0 0 1-4 4H7"></path>
  </svg>
  <svg v-else-if="name === 'bitbucket'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
    <path d="M5 5h14l-2 14H7z"></path>
    <path d="M9 9h6"></path>
    <path d="M10 9l1 7"></path>
  </svg>
  <svg v-else-if="name === 'confluence'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
    <path d="M6 16c3-5 5-8 12-8"></path>
    <path d="M8 18c2-4 4-6 10-6"></path>
  </svg>
  <svg v-else-if="name === 'microservices'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
    <path d="M12 3l9 4.5v9L12 21l-9-4.5v-9z"></path>
    <path d="M3 7.5 12 12l9-4.5"></path>
    <path d="M12 12v9"></path>
  </svg>
  <svg v-else-if="name === 'copilot'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
    <path d="M6 8a6 6 0 0 1 12 0v6a4 4 0 0 1-4 4H10a4 4 0 0 1-4-4z"></path>
    <path d="M9 11h.01"></path>
    <path d="M15 11h.01"></path>
    <path d="M9 15c2 1 4 1 6 0"></path>
  </svg>
  <svg v-else-if="name === 'settings'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
    <path d="M12 3v2"></path>
    <path d="M12 19v2"></path>
    <path d="m4.93 4.93 1.41 1.41"></path>
    <path d="m17.66 17.66 1.41 1.41"></path>
    <path d="M3 12h2"></path>
    <path d="M19 12h2"></path>
    <path d="m4.93 19.07 1.41-1.41"></path>
    <path d="m17.66 6.34 1.41-1.41"></path>
    <circle cx="12" cy="12" r="4"></circle>
  </svg>
</template>

<style scoped>
svg {
  display: block;
  width: 100%;
  height: 100%;
}
</style>
