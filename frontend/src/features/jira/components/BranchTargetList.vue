<script setup lang="ts">
import { computed } from "vue"
import type { IssueBranchMatch } from "../branch-types"
import { useBranchCreation } from "../composables/useBranchCreation"
import { type BranchPlanState } from "../logic/branch-plan"
import { elsewhereMatches } from "../logic/branch-plan-progress"
import BranchTargetRow from "./BranchTargetRow.vue"
import RepositoryPicker from "./RepositoryPicker.vue"

const props = defineProps<{
  plan: BranchPlanState
  issueKey: string
  isCreating: boolean
  matches: IssueBranchMatch[]
  isLoadingSuggestions: boolean
  suggestionError: string
  unavailableReasons: string[]
}>()

const emit = defineEmits<{
  (event: "retry-suggestions"): void
}>()

const { addRepository } = useBranchCreation()

const takenRepos = computed(() => props.plan.targets.map((target) => target.repo))
const suggestedRepos = computed(() =>
  props.plan.targets.filter((target) => !target.isManual).map((target) => target.repo)
)
const isSuggestionEmpty = computed(
  () => suggestedRepos.value.length === 0 && props.plan.targets.length === 0
)
const elsewhereRepos = computed(() => {
  const repos: string[] = []
  for (const match of elsewhereMatches(props.plan, props.matches)) {
    if (!repos.includes(match.repo)) {
      repos.push(match.repo)
    }
  }
  return repos
})
const add = (repo: string, isCloned: boolean): void => {
  void addRepository(props.issueKey, repo, isCloned)
}
</script>

<template>
  <section class="target-list" data-testid="branch-target-list">
    <h2 class="target-list-title">Where this will be built</h2>

    <p v-if="isLoadingSuggestions" class="target-status">Looking for the repositories this issue touches…</p>

    <div v-else-if="suggestionError.length > 0" class="target-status is-failed">
      <span>{{ suggestionError }}</span>
      <button
        class="link-btn"
        type="button"
        data-testid="branch-suggestion-retry"
        @click="emit('retry-suggestions')"
      >
        Retry
      </button>
    </div>

    <p v-else-if="isSuggestionEmpty" class="empty" data-testid="branch-suggestion-empty">
      Lunar could not suggest a repository from the acceptance criteria. Choose one yourself.
    </p>

    <BranchTargetRow
      v-for="target in plan.targets"
      :key="target.repo"
      :plan="plan"
      :issue-key="issueKey"
      :repo="target.repo"
      :is-creating="isCreating"
    />

    <p v-if="elsewhereRepos.length > 0" class="notice" data-testid="branch-existing-elsewhere">
      A branch for this issue also exists in {{ elsewhereRepos.join(", ") }}, which
      {{ elsewhereRepos.length > 1 ? "are" : "is" }} not in this list.
      <button
        v-for="repo in elsewhereRepos"
        :key="repo"
        class="link-btn"
        type="button"
        :data-testid="`branch-existing-elsewhere-add-${repo}`"
        @click="add(repo, true)"
      >
        Add {{ repo }}
      </button>
    </p>

    <RepositoryPicker :taken-repos="takenRepos" @add="add" />

    <div v-if="unavailableReasons.length > 0" class="unavailable" data-testid="branch-signals-unavailable">
      <p v-for="reason in unavailableReasons" :key="reason" class="unavailable-line">{{ reason }}</p>
    </div>
  </section>
</template>

<style scoped src="./branch-target-list.css"></style>
