<script setup lang="ts">
import { computed } from "vue"
import { useBaseBranchOptions } from "../composables/useBaseBranchOptions"
import { useBranchCreation } from "../composables/useBranchCreation"
import { editBranchName, type BranchPlanState } from "../logic/branch-plan"
import {
  creatableTargets,
  isBranchNameLocked,
  missingBaseTargets,
  planSummary,
  primaryAction
} from "../logic/branch-plan-progress"

const props = defineProps<{
  plan: BranchPlanState
  issueKey: string
  isLoadingName: boolean
  nameError: string
}>()

const emit = defineEmits<{
  (event: "retry-name"): void
}>()

const { optionsFor } = useBaseBranchOptions()
const { createFor, createErrorFor, isCreatingFor, feedbackError } = useBranchCreation()

const isCreating = computed(() => isCreatingFor(props.issueKey))
const action = computed(() => primaryAction(props.plan))
const summary = computed(() => planSummary(props.plan))
const isLocked = computed(() => isBranchNameLocked(props.plan))
const createError = computed(() => createErrorFor(props.issueKey))
const missingBaseCount = computed(() => missingBaseTargets(props.plan).length)
const creatableRepos = computed(() => creatableTargets(props.plan).map((target) => target.repo))

const selectedSource = (repo: string): string => {
  const base = props.plan.bases[repo] ?? ""
  const option = optionsFor(repo)?.options.find((entry) => entry.name === base) ?? null
  return option?.source ?? ""
}

const crossTeamRepos = computed(() =>
  creatableRepos.value.filter((repo) => selectedSource(repo) === "other")
)

const notClonedRepos = computed(() =>
  creatableRepos.value.filter((repo) => optionsFor(repo)?.is_cloned === false)
)

const evidenceText = computed(() => {
  if (props.isLoadingName) {
    return "Reading the acceptance criteria for a branch name…"
  }
  if (props.plan.branchNameEvidence.length > 0) {
    return props.plan.branchNameEvidence
  }
  return ""
})

const evidencePrefix = computed(() => {
  if (props.plan.branchNameSource === "model") {
    return "Read from the acceptance criteria:"
  }
  if (props.plan.branchNameSource === "fallback") {
    return "The model was not asked, so this came from"
  }
  return ""
})

const evidenceSuffix = computed(() =>
  props.plan.branchNameSource === "fallback" ? "Ask the model to improve it." : ""
)

const updateName = (event: Event): void => {
  if (!(event.target instanceof HTMLInputElement)) {
    return
  }
  editBranchName(props.plan, event.target.value)
}

const submit = (): void => {
  void createFor(props.issueKey)
}
</script>

<template>
  <section class="create-form" data-testid="branch-create-form">
    <h2 class="create-title">Branch name</h2>

    <input
      class="field mono"
      type="text"
      aria-label="Branch name"
      data-testid="branch-name-input"
      :value="plan.branchName"
      :disabled="isLocked || isLoadingName"
      @input="updateName"
    />

    <div v-if="isLocked" class="derived" data-testid="branch-name-locked">
      The name is fixed because branches already exist with it. Changing it now would leave the created
      branches inconsistent.
    </div>

    <div v-else-if="nameError.length > 0" class="failure" data-testid="branch-name-error">
      <span>{{ nameError }}</span>
      <button class="link-btn" type="button" @click="emit('retry-name')">Retry</button>
    </div>

    <div v-else-if="evidenceText.length > 0" class="derived" data-testid="branch-name-evidence">
      <template v-if="evidencePrefix.length > 0">
        {{ evidencePrefix }} <b>{{ evidenceText }}</b>
      </template>
      <template v-else>{{ evidenceText }}</template>
      <template v-if="evidenceSuffix.length > 0"> {{ evidenceSuffix }}</template>
    </div>

    <div class="actions">
      <button
        class="primary"
        type="button"
        data-testid="branch-create"
        :disabled="!action.enabled || isCreating"
        @click="submit"
      >
        {{ isCreating ? "Creating…" : action.label }}
      </button>
      <span v-if="missingBaseCount > 0" class="hint">
        Choose a base branch for every repository you keep
      </span>
    </div>

    <div
      v-if="summary !== null"
      class="summary"
      :class="summary.tone === 'partial' ? 'is-partial' : 'is-ok'"
      data-testid="branch-create-summary"
    >
      <b>{{ summary.countLabel }}</b> {{ summary.stateLabel }}.
      <template v-if="summary.detail.length > 0"> {{ summary.detail }}</template>
    </div>

    <div v-if="createError.length > 0" class="failure" data-testid="branch-create-error">
      {{ createError }}
    </div>

    <div v-if="crossTeamRepos.length > 0" class="cross-note" data-testid="branch-cross-team-note">
      {{ crossTeamRepos.join(", ") }} will branch from another team's branch, which has not reached
      production yet. Your branch carries that dependency until theirs is promoted.
    </div>

    <div v-if="notClonedRepos.length > 0" class="warn-note" data-testid="branch-not-cloned-note">
      {{ notClonedRepos.join(", ") }}
      {{ notClonedRepos.length > 1 ? "are" : "is" }} not cloned.
      {{ notClonedRepos.length > 1 ? "Those branches" : "That branch" }} will fail unless
      {{ notClonedRepos.length > 1 ? "they are" : "it is" }} cloned first.
    </div>

    <div v-if="feedbackError.length > 0" class="feedback-error" role="status" data-testid="branch-feedback-error">
      {{ feedbackError }}
    </div>
  </section>
</template>

<style scoped src="./branch-create-form.css"></style>
