package infrastructure

import (
	"strings"

	"lunar/backend/internal/jira/domain"
)

func enrichIssueMetadata(issue *domain.JiraIssue) {
	applySprintMetadata(issue)

	key := strings.ToUpper(issue.Key)
	summary := strings.ToLower(issue.Fields.Summary)
	var typeName string
	if issue.Fields.IssueType != nil {
		typeName = strings.ToLower(issue.Fields.IssueType.Name)
	}

	if strings.HasPrefix(key, "SOP-") ||
		strings.Contains(summary, "dokumen sop") ||
		strings.Contains(summary, "sop operasional") ||
		strings.Contains(summary, "[sop]") ||
		strings.Contains(summary, " sop ") ||
		strings.HasPrefix(summary, "sop ") ||
		strings.HasSuffix(summary, " sop") {
		issue.Kind = "sop"
		issue.SubLabel = "DEV Document · SOP"
		if issue.Points == 0 {
			issue.Points = 1
		}
		return
	}

	if strings.HasPrefix(key, "UT-") ||
		strings.Contains(summary, "dokumen ut") ||
		strings.Contains(summary, "unit test") ||
		strings.Contains(summary, "[ut]") ||
		strings.Contains(summary, " ut ") ||
		strings.HasPrefix(summary, "ut ") ||
		strings.HasSuffix(summary, " ut") ||
		typeName == "ut" {
		issue.Kind = "ut"
		issue.SubLabel = "DEV Document · UT"
		if issue.Points == 0 {
			issue.Points = 2
		}
		return
	}

	if strings.HasPrefix(key, "QR-") ||
		strings.Contains(summary, "query review") ||
		strings.Contains(summary, "review query") ||
		strings.Contains(summary, "[query]") ||
		strings.Contains(summary, "[qr]") ||
		strings.Contains(summary, "query ") ||
		strings.HasPrefix(summary, "query") {
		issue.Kind = "query"
		issue.SubLabel = "DEV Document · Query"
		if issue.Points == 0 {
			issue.Points = 3
		}
		return
	}

	if strings.Contains(typeName, "documentation") ||
		strings.Contains(summary, "dokumentasi") ||
		strings.Contains(summary, "dokumen ") ||
		strings.HasPrefix(summary, "dokumen") {
		issue.Kind = "ut"
		issue.SubLabel = "DEV Document"
		if issue.Points == 0 {
			issue.Points = 2
		}
		return
	}

	if strings.HasPrefix(key, "BUG-") ||
		strings.Contains(typeName, "bug") ||
		strings.Contains(typeName, "defect") ||
		strings.Contains(summary, "bug ") ||
		strings.Contains(summary, "defect ") {
		issue.Kind = "bug"
		issue.SubLabel = "Bug Fix"
		if issue.Points == 0 {
			issue.Points = 3
		}
		return
	}

	if strings.HasPrefix(key, "SUB-") ||
		(issue.Fields.IssueType != nil && issue.Fields.IssueType.Subtask) ||
		strings.Contains(typeName, "sub") {
		issue.Kind = "subtask"
		issue.SubLabel = "Subtask"
		if issue.Points == 0 {
			issue.Points = 2
		}
		return
	}

	if strings.Contains(typeName, "story") {
		issue.Kind = "story"
		issue.SubLabel = "MMS Story"
		return
	}

	issue.Kind = "task"
	issue.SubLabel = "MMS Task"
	if issue.Points == 0 {
		issue.Points = 3
	}
}

func applySprintMetadata(issue *domain.JiraIssue) {
	for _, rawSprint := range issue.Fields.SprintValues {
		reference := domain.ParseSprintReference(rawSprint)
		if issue.SprintName == "" {
			issue.SprintName = reference.Name
		}
		if issue.SprintBoardID == 0 {
			issue.SprintBoardID = reference.BoardID
		}
		if issue.SprintName != "" && issue.SprintBoardID != 0 {
			return
		}
	}
}
