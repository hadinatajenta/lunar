<script setup lang="ts">
import { computed } from "vue"
import { useIssueBranches } from "../composables/useIssueBranches"
import { useBranchCreation } from "../composables/useBranchCreation"
import { issueBranchPill } from "../logic/issue-branch-pill"

const props = defineProps<{
  issueKey: string
}>()

const { branchesForIssue, sprintMatchesLoaded } = useIssueBranches()
const { progressFor } = useBranchCreation()

const pill = computed(() => {
  const progress = progressFor(props.issueKey)
  if (progress === null && !sprintMatchesLoaded.value) {
    return null
  }
  return issueBranchPill(branchesForIssue(props.issueKey), progress)
})
</script>

<template>
  <span
    v-if="pill !== null"
    class="issue-branch-pill"
    :class="`is-${pill.tone}`"
    data-testid="issue-branch-pill"
  >
    {{ pill.text }}
  </span>
</template>

<style scoped src="./issue-branch-pill.css"></style>
