<template>
  <RouterView />
  <Toast />
</template>

<script setup lang="ts">
import { watch } from "vue"
import { RouterView } from "vue-router"
import Toast from "../components/ui/Toast.vue"
import { useAuth } from "../features/auth/composables/useAuth"
import { useJira } from "../features/jira/composables/useJira"
import { useConfluence } from "../features/confluence/composables/useConfluence"

const { isAuthenticated, fetchUser } = useAuth()
const { prefetchJiraData, resetJiraData } = useJira()
const { resetConfluenceData } = useConfluence()

watch(
  isAuthenticated,
  (authenticated) => {
    if (authenticated) {
      fetchUser()
      prefetchJiraData()
    } else {
      resetJiraData()
      resetConfluenceData()
    }
  },
  { immediate: true }
)
</script>
