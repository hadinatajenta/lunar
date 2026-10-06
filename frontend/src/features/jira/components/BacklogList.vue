<script setup lang="ts">
import { computed } from "vue"
import type { JiraIssue, JiraSprint } from "../types"
import BacklogSprint from "./BacklogSprint.vue"

interface Props {
  sprints: JiraSprint[]
  squadName?: string
}

const props = withDefaults(defineProps<Props>(), {
  squadName: ""
})

const emit = defineEmits<{
  (e: "select-issue", issue: JiraIssue): void
}>()

const hasSquad = computed(() => props.squadName.trim().length > 0)

const squadLabel = computed(() =>
  hasSquad.value ? props.squadName.trim() : "No squad could be detected for your account"
)
</script>

<template>
  <div class="backlog-container" data-testid="backlog-container">
    <div class="backlog-header" data-testid="backlog-header">
      <span class="backlog-header-label">Squad</span>
      <strong
        :class="['backlog-header-value', { 'is-empty': !hasSquad }]"
        data-testid="backlog-squad-label"
      >
        {{ squadLabel }}
      </strong>
    </div>

    <div
      v-if="sprints.length === 0"
      class="backlog-empty"
      data-testid="backlog-empty"
    >
      No sprints available for this squad
    </div>

    <BacklogSprint
      v-for="sprint in sprints"
      :key="sprint.id"
      :sprint="sprint"
      @select-issue="emit('select-issue', $event)"
    />
  </div>
</template>

<style scoped>
.backlog-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.backlog-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface-raised);
}

.backlog-header-label {
  color: var(--subtle);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.09em;
  text-transform: uppercase;
}

.backlog-header-value {
  color: var(--text);
  font-size: 12.5px;
  font-weight: 600;
}

.backlog-header-value.is-empty {
  color: var(--muted);
  font-weight: 500;
}

.backlog-empty {
  padding: 48px 24px;
  text-align: center;
  border: 1px dashed var(--border);
  border-radius: 12px;
  color: var(--muted);
  font-size: 13px;
}
</style>
