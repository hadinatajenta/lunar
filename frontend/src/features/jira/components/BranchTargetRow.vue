<script setup lang="ts">
import { computed } from "vue"
import { useBaseBranchOptions } from "../composables/useBaseBranchOptions"
import { useBranchCreation } from "../composables/useBranchCreation"
import {
  isTargetExcluded,
  reconsiderExistingBranch,
  resolveExistingBranch,
  setBaseBranch,
  setRememberBase,
  setTargetExcluded,
  type BranchPlanState
} from "../logic/branch-plan"
import { rowKind } from "../logic/branch-plan-progress"
import BaseBranchPicker from "./BaseBranchPicker.vue"

const props = defineProps<{
  plan: BranchPlanState
  issueKey: string
  repo: string
  isCreating: boolean
}>()

const { optionsFor, errorFor, isLoadingFor, loadBaseBranchOptions } = useBaseBranchOptions()
const { retryTarget, removeRepository, recordFeedback } = useBranchCreation()

const kind = computed(() => rowKind(props.plan, props.repo))
const target = computed(() => props.plan.targets.find((entry) => entry.repo === props.repo) ?? null)
const outcome = computed(() => props.plan.outcomes[props.repo] ?? null)
const existing = computed(() => props.plan.existing[props.repo] ?? null)
const base = computed(() => props.plan.bases[props.repo] ?? "")
const reasons = computed(() => target.value?.reasons ?? [])
const isManual = computed(() => target.value?.isManual === true)
const isIncluded = computed(() => !isTargetExcluded(props.plan, props.repo))
const isRemembered = computed(() => props.plan.rememberBases[props.repo] === true)
const isCloned = computed(() => {
  const options = optionsFor(props.repo)
  if (options !== null) {
    return options.is_cloned
  }
  return target.value?.isCloned === true
})
const existingLabel = computed(() => {
  if (existing.value === null) {
    return ""
  }
  return existing.value.isCloned ? existing.value.branch : `${existing.value.branch} · on Bitbucket, not cloned`
})
const duplicateWarning = computed(() => {
  if (existing.value === null) {
    return ""
  }
  return `${existing.value.branch} already exists here. Creating another splits the work for this issue across two branches.`
})

const loadOptions = (): void => {
  void loadBaseBranchOptions(props.repo)
}

const selectBase = (selectedBase: string): void => {
  setBaseBranch(props.plan, props.repo, selectedBase)
}

const resolveUse = (): void => {
  resolveExistingBranch(props.plan, props.repo, "use")
}

const resolveNew = (): void => {
  resolveExistingBranch(props.plan, props.repo, "new")
}

const reconsider = (): void => {
  reconsiderExistingBranch(props.plan, props.repo)
}

const includeTarget = (event: Event): void => {
  const isChecked = event.target instanceof HTMLInputElement ? event.target.checked : true
  setTargetExcluded(props.plan, props.repo, !isChecked)
  void recordFeedback(props.issueKey, props.repo, isChecked ? "kept" : "unticked")
}

const rememberBase = (event: Event): void => {
  const isChecked = event.target instanceof HTMLInputElement ? event.target.checked : false
  setRememberBase(props.plan, props.repo, isChecked)
}

const remove = (): void => {
  void removeRepository(props.issueKey, props.repo)
}

const retry = (): void => {
  void retryTarget(props.issueKey, props.repo)
}
</script>

<template>
  <div
    class="target"
    :class="{
      'is-outcome': kind === 'outcome',
      'is-ok': kind === 'existing-same-name' || kind === 'existing-kept' || outcome?.status === 'created' || outcome?.status === 'exists',
      'is-failed': outcome?.status === 'failed',
      'is-conflict': kind === 'conflict',
      'is-off': kind !== 'outcome' && !isIncluded
    }"
    :data-testid="`branch-target-${repo}`"
    :data-state="kind"
  >
    <template v-if="kind === 'outcome' && outcome !== null">
      <span class="mark">{{ outcome.status === 'failed' ? '✕' : '✓' }}</span>
      <div class="target-main">
        <div class="repo">{{ repo }}</div>
        <div v-if="outcome.status === 'created'" class="sub-line">Created</div>
        <div v-else-if="outcome.status === 'exists'" class="sub-line">Already existed, left alone</div>
        <div v-else class="sub-line is-reason">{{ outcome.reason }}</div>
        <div v-if="outcome.status === 'created'" class="sub-line mono-line">
          {{ outcome.branch }} from {{ outcome.baseBranch }}
        </div>
        <div v-else-if="outcome.status === 'exists'" class="sub-line mono-line">{{ outcome.branch }}</div>
      </div>
      <div class="outcome-side">
        <button
          v-if="outcome.status === 'failed'"
          class="retry"
          type="button"
          :disabled="isCreating"
          :data-testid="`branch-target-retry-${repo}`"
          @click="retry"
        >
          Retry
        </button>
      </div>
    </template>

    <template v-else-if="kind === 'existing-same-name' && existing !== null">
      <span class="mark">✓</span>
      <div class="target-main">
        <div class="repo">{{ repo }}</div>
        <div class="sub-line">Already exists with this name, left alone</div>
        <div class="sub-line mono-line">{{ existingLabel }}</div>
      </div>
      <div class="outcome-side"></div>
    </template>

    <template v-else-if="kind === 'existing-kept' && existing !== null">
      <span class="mark">✓</span>
      <div class="target-main">
        <div class="repo">{{ repo }}</div>
        <div class="sub-line">Using the existing branch</div>
        <div class="sub-line mono-line">{{ existing.branch }}</div>
      </div>
      <div class="outcome-side">
        <button
          class="retry"
          type="button"
          :data-testid="`branch-target-change-${repo}`"
          @click="reconsider"
        >
          Change
        </button>
      </div>
    </template>

    <template v-else-if="kind === 'conflict' && existing !== null">
      <span class="mark">!</span>
      <div class="target-main">
        <div class="repo">{{ repo }}</div>
        <div class="sub-line is-reason">A branch for this issue already exists here, under a different name.</div>
        <div class="sub-line mono-line">{{ existing.branch }}</div>
        <div class="sub-line">The name Lunar would create is {{ plan.branchName }}.</div>
      </div>
      <div class="outcome-side">
        <button
          class="retry"
          type="button"
          :data-testid="`branch-target-use-existing-${repo}`"
          @click="resolveUse"
        >
          Use the existing one
        </button>
        <button
          class="ghost small"
          type="button"
          :data-testid="`branch-target-create-anyway-${repo}`"
          @click="resolveNew"
        >
          Create a new one anyway
        </button>
      </div>
    </template>

    <template v-else>
      <span v-if="isManual" class="mark-slot" aria-hidden="true"></span>
      <input
        v-else
        class="target-include"
        type="checkbox"
        :checked="isIncluded"
        :aria-label="`Include ${repo}`"
        :data-testid="`branch-target-include-${repo}`"
        @change="includeTarget"
      />
      <div class="target-main">
        <div class="repo">
          {{ repo }}
          <span v-if="!isCloned" class="tag">not cloned</span>
        </div>
        <div v-for="reason in reasons" :key="reason" class="sub-line">{{ reason }}</div>
        <div v-if="kind === 'pending-duplicate'" class="sub-line is-reason">{{ duplicateWarning }}</div>
      </div>
      <div class="target-side">
        <BaseBranchPicker
          :repo="repo"
          :selected="base"
          :options="optionsFor(repo)"
          :is-loading="isLoadingFor(repo)"
          :error-message="errorFor(repo)"
          @select="selectBase"
          @load="loadOptions"
        />
        <label class="remember-line">
          <input
            class="remember-input"
            type="checkbox"
            :checked="isRemembered"
            :data-testid="`branch-target-remember-${repo}`"
            @change="rememberBase"
          />
          <span>remember this base for {{ repo }}</span>
        </label>
        <button
          v-if="isManual"
          class="remove"
          type="button"
          :aria-label="`Remove ${repo}`"
          :data-testid="`branch-target-remove-${repo}`"
          @click="remove"
        >
          ×
        </button>
      </div>
    </template>
  </div>
</template>

<style scoped src="./branch-target-row.css"></style>
