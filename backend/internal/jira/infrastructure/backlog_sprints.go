package infrastructure

import (
	"lunar/backend/internal/jira/domain"
)

func filterBacklogSprints(sprintList []domain.JiraSprint) []domain.JiraSprint {
	filteredSprints := make([]domain.JiraSprint, 0, len(sprintList))
	for _, sprint := range sprintList {
		if sprint.OriginBoardID > 0 && sprint.OriginBoardID != JIRA_BACKLOG_BOARD_ID {
			continue
		}
		filteredSprints = append(filteredSprints, sprint)
	}
	return filteredSprints
}

func selectPrioritySprints(sprints []domain.JiraSprint) []domain.JiraSprint {
	keptIndices := make(map[int]bool, len(sprints))
	for _, groupIndices := range groupSprintIndicesBySquad(sprints) {
		for _, index := range pickPriorityGroupIndices(groupIndices, sprints) {
			keptIndices[index] = true
		}
	}

	prioritySprints := make([]domain.JiraSprint, 0, len(keptIndices))
	for index, sprint := range sprints {
		if keptIndices[index] {
			prioritySprints = append(prioritySprints, sprint)
		}
	}
	return prioritySprints
}

func groupSprintIndicesBySquad(sprints []domain.JiraSprint) [][]int {
	squadOrder := make([]string, 0, len(sprints))
	squadGroups := make(map[string][]int, len(sprints))
	for index, sprint := range sprints {
		squad := domain.ExtractSquadName(sprint.Name)
		if _, exists := squadGroups[squad]; !exists {
			squadOrder = append(squadOrder, squad)
		}
		squadGroups[squad] = append(squadGroups[squad], index)
	}

	groups := make([][]int, 0, len(squadOrder))
	for _, squad := range squadOrder {
		groups = append(groups, squadGroups[squad])
	}
	return groups
}

func pickPriorityGroupIndices(groupIndices []int, sprints []domain.JiraSprint) []int {
	activeIndices := make([]int, 0, len(groupIndices))
	for _, index := range groupIndices {
		if sprints[index].State == "active" {
			activeIndices = append(activeIndices, index)
		}
	}
	if len(activeIndices) > 0 {
		return activeIndices
	}
	return groupIndices[:1]
}
