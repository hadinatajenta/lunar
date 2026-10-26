<script setup lang="ts">
import { computed, watch } from "vue"
import type { JiraIssue } from "../types"
import { useBranchCreation } from "../composables/useBranchCreation"
import { useBranchName } from "../composables/useBranchName"
import { useIssueBranchMatches } from "../composables/useIssueBranchMatches"
import { useRepositorySuggestion } from "../composables/useRepositorySuggestion"
import {
  applyBranchName,
  seedExistingBranches,
  seedSuggestedTargets
} from "../logic/branch-plan"
import BranchCreateForm from "./BranchCreateForm.vue"
import BranchTargetList from "./BranchTargetList.vue"
import IssueBranchList from "./IssueBranchList.vue"

const props = defineProps<{
  issue: JiraIssue
}>()

const { matchesFor, scopeFor, errorFor, isLoadingFor, loadIssueBranchMatches } = useIssueBranchMatches()
const {
  suggestionsFor,
  confidenceFor,
  unavailableFor,
  errorFor: suggestionErrorFor,
  isLoadingFor: isSuggestionLoadingFor,
  loadSuggestions
} = useRepositorySuggestion()
const { nameFor, errorFor: nameErrorFor, isLoadingFor: isNameLoadingFor, loadBranchName } = useBranchName()
const { planFor, isCreatingFor } = useBranchCreation()

const issueKey = computed(() => props.issue.key || props.issue.id)
const matches = computed(() => matchesFor(issueKey.value))
const scope = computed(() => scopeFor(issueKey.value))
const matchesError = computed(() => errorFor(issueKey.value))
const isLoadingMatches = computed(() => isLoadingFor(issueKey.value))
const suggestionError = computed(() => suggestionErrorFor(issueKey.value))
const isLoadingSuggestions = computed(() => isSuggestionLoadingFor(issueKey.value))
const nameError = computed(() => nameErrorFor(issueKey.value))
const isLoadingName = computed(() => isNameLoadingFor(issueKey.value))
const unavailableReasons = computed(() => unavailableFor(issueKey.value))
const confidence = computed(() => confidenceFor(issueKey.value))
const plan = computed(() => planFor(issueKey.value))
const isCreating = computed(() => isCreatingFor(issueKey.value))

const seedPlan = (): void => {
  const currentPlan = planFor(issueKey.value)
  seedSuggestedTargets(currentPlan, suggestionsFor(issueKey.value), confidenceFor(issueKey.value))
  const suggestion = nameFor(issueKey.value)
  if (suggestion !== null) {
    applyBranchName(currentPlan, suggestion)
  }
  seedExistingBranches(currentPlan, matchesFor(issueKey.value))
}

const load = async (): Promise<void> => {
  const key = issueKey.value
  if (key.length === 0) {
    return
  }
  seedPlan()
  await Promise.all([
    loadIssueBranchMatches(key),
    loadSuggestions(key),
    loadBranchName(key)
  ])
  seedPlan()
}

const retryMatches = (): void => {
  void loadIssueBranchMatches(issueKey.value, { force: true })
}

const retrySuggestions = (): void => {
  void loadSuggestions(issueKey.value, { force: true })
}

const retryName = (): void => {
  void loadBranchName(issueKey.value, { force: true })
}

watch(
  [
    issueKey,
    () => matchesFor(issueKey.value),
    () => suggestionsFor(issueKey.value),
    () => confidenceFor(issueKey.value),
    () => nameFor(issueKey.value)
  ],
  seedPlan
)

watch(issueKey, load, { immediate: true })
</script>

<template>
  <div class="branch-panel" data-testid="issue-branch-panel">
    <IssueBranchList
      :issue-key="issueKey"
      :matches="matches"
      :scope="scope"
      :is-loading="isLoadingMatches"
      :error-message="matchesError"
      @retry="retryMatches"
    />

    <BranchTargetList
      :plan="plan"
      :issue-key="issueKey"
      :is-creating="isCreating"
      :matches="matches"
      :is-loading-suggestions="isLoadingSuggestions"
      :suggestion-error="suggestionError"
      :confidence="confidence"
      :unavailable-reasons="unavailableReasons"
      @retry-suggestions="retrySuggestions"
    />

    <BranchCreateForm
      v-if="plan.targets.length > 0"
      :plan="plan"
      :issue-key="issueKey"
      :is-loading-name="isLoadingName"
      :name-error="nameError"
      @retry-name="retryName"
    />
  </div>
</template>

<style scoped src="./issue-branch-panel.css"></style>
