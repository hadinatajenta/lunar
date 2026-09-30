package infrastructure

import (
	"strings"

	"lunar/backend/internal/jira/domain"
)

func buildSeedIssues() []domain.JiraIssue {
	projectMMS := domain.JiraProject{
		ID:   "10001",
		Key:  "CRMMS",
		Name: "Customer Relationship Management Merchant System",
	}

	userAssignee := &domain.JiraUser{
		Self:         "https://jira.bri.co.id/rest/api/2/user?username=developer",
		Name:         "developer",
		Key:          "developer",
		EmailAddress: "developer@lunar.dev",
		DisplayName:  "Lunar Developer (BRI MMS)",
	}

	parent77911 := &domain.JiraParent{
		ID:   "10101",
		Key:  "CRMMS-77911",
		Self: "https://jira.bri.co.id/rest/api/2/issue/10101",
		Fields: domain.JiraParentFields{
			Summary: "Merchant Onboarding Multi-Tier Verification Service",
		},
	}

	parent77913 := &domain.JiraParent{
		ID:   "10102",
		Key:  "CRMMS-77913",
		Self: "https://jira.bri.co.id/rest/api/2/issue/10102",
		Fields: domain.JiraParentFields{
			Summary: "MMS Settlement Batch Processing Pipeline Optimization",
		},
	}

	parent77914 := &domain.JiraParent{
		ID:   "10103",
		Key:  "CRMMS-77914",
		Self: "https://jira.bri.co.id/rest/api/2/issue/10103",
		Fields: domain.JiraParentFields{
			Summary: "QRIS Merchant Aggregator Dynamic Fee Calculation",
		},
	}

	parent77925 := &domain.JiraParent{
		ID:   "10104",
		Key:  "CRMMS-77925",
		Self: "https://jira.bri.co.id/rest/api/2/issue/10104",
		Fields: domain.JiraParentFields{
			Summary: "Dispute & Chargeback Resolution Workflow Integration",
		},
	}

	parent77926 := &domain.JiraParent{
		ID:   "10105",
		Key:  "CRMMS-77926",
		Self: "https://jira.bri.co.id/rest/api/2/issue/10105",
		Fields: domain.JiraParentFields{
			Summary: "High-Throughput Push Notification Worker for Merchant Portal",
		},
	}

	return []domain.JiraIssue{
		createIssue("10101", "CRMMS-77911", "Merchant Onboarding Multi-Tier Verification Service", "Automated KYC verification and document scoring pipeline.", "story", 8, "MMS Core", "3", "In Progress", "yellow", "Story", false, "2", "High", projectMMS, userAssignee, nil, "Sprint 44 - Fortune Squad"),
		createIssue("10102", "CRMMS-77913", "MMS Settlement Batch Processing Pipeline Optimization", "Optimize end-of-day reconciliation queries and concurrent batch execution routines.", "task", 5, "MMS Core", "4", "In Review", "yellow", "Task", false, "2", "High", projectMMS, userAssignee, nil, "Sprint 44 - Fortune Squad"),
		createIssue("10103", "CRMMS-77914", "QRIS Merchant Aggregator Dynamic Fee Calculation", "Support tiered MDR calculation matrix based on transaction volume thresholds.", "story", 5, "QRIS Engine", "3", "In Progress", "yellow", "Story", false, "2", "High", projectMMS, userAssignee, nil, "Sprint 44 - Fortune Squad"),
		createIssue("10104", "CRMMS-77925", "Dispute & Chargeback Resolution Workflow Integration", "Wire dispute status transition webhooks to upstream core banking notification service.", "task", 3, "Dispute Mgmt", "1", "To Do", "blue-gray", "Task", false, "3", "Medium", projectMMS, userAssignee, nil, "Sprint 45 - Fortune Squad"),
		createIssue("10105", "CRMMS-77926", "High-Throughput Push Notification Worker for Merchant Portal", "Implement async worker queue processing outbound alerts for transaction events.", "task", 3, "Notification", "1", "To Do", "blue-gray", "Task", false, "3", "Medium", projectMMS, userAssignee, nil, "Sprint 45 - Fortune Squad"),
		createIssue("10201", "BUG-201", "Race condition during concurrent QRIS settlement callback processing", "Duplicate ledger entries generated when webhook retried within 100ms window.", "bug", 3, "Bug Fix", "3", "In Progress", "yellow", "Bug", false, "1", "Highest", projectMMS, userAssignee, nil, "Sprint 44 - Fortune Squad"),
		createIssue("10202", "BUG-202", "Nil pointer dereference on merchant profile with null NPWP field", "Fix unhandled nullable tax identification number in validation parser.", "bug", 2, "Bug Fix", "5", "Done", "green", "Bug", false, "2", "High", projectMMS, userAssignee, nil, "Sprint 44 - Fortune Squad"),
		createIssue("10203", "BUG-203", "Memory leak in transaction log stream WebSocket connection pool", "Zombie client sessions not evicted upon TCP RST disconnect.", "bug", 5, "Bug Fix", "1", "To Do", "blue-gray", "Bug", false, "2", "High", projectMMS, userAssignee, nil, "Sprint 45 - Fortune Squad"),
		createIssue("10301", "SUB-101", "Implement database migration for merchant KYC verification tables", "Create indexed tables for KYC multi-stage verification status.", "subtask", 2, "MMS Core", "5", "Done", "green", "Sub-task", true, "3", "Medium", projectMMS, userAssignee, parent77911, "Sprint 44 - Fortune Squad"),
		createIssue("10302", "SUB-102", "Create mock verification provider client for local sandbox testing", "Provide deterministic mock responses for third-party KYC checks.", "subtask", 2, "MMS Core", "3", "In Progress", "yellow", "Sub-task", true, "3", "Medium", projectMMS, userAssignee, parent77911, "Sprint 44 - Fortune Squad"),
		createIssue("10303", "SUB-103", "Add audit logging for merchant status transitions", "Structured audit log stream for compliance auditing.", "subtask", 1, "MMS Core", "1", "To Do", "blue-gray", "Sub-task", true, "4", "Low", projectMMS, userAssignee, parent77911, "Sprint 44 - Fortune Squad"),
		createIssue("10401", "UT-101", "Generate Unit Test document and test suite for Settlement Service", "Prepare hermetic unit tests and automated test evidence documentation.", "ut", 2, "UT Docs", "3", "In Progress", "yellow", "Sub-task", true, "2", "High", projectMMS, userAssignee, parent77913, "Sprint 44 - Fortune Squad"),
		createIssue("10402", "UT-102", "Generate Unit Test document for QRIS Fee Calculator matrix", "Coverage for boundary fees and rounding edge cases.", "ut", 2, "UT Docs", "1", "To Do", "blue-gray", "Sub-task", true, "3", "Medium", projectMMS, userAssignee, parent77914, "Sprint 44 - Fortune Squad"),
		createIssue("10403", "UT-103", "Generate Unit Test document for Dispute State Machine transitions", "State transition invariants and dispute rejection edge tests.", "ut", 2, "UT Docs", "1", "To Do", "blue-gray", "Sub-task", true, "3", "Medium", projectMMS, userAssignee, parent77925, "Sprint 45 - Fortune Squad"),
		createIssue("10404", "UT-104", "Unit test coverage for FCM token refresh retry backoff", "Deterministic fake timer tests for exponential backoff retry.", "ut", 1, "UT Docs", "5", "Done", "green", "Sub-task", true, "4", "Low", projectMMS, userAssignee, parent77926, "Sprint 45 - Fortune Squad"),
		createIssue("10501", "QR-201", "Query Review: optimize merchant transaction history indexing", "EXPLAIN ANALYZE index inspection on merchant transaction range queries.", "query", 3, "Query Review", "4", "In Review", "yellow", "Task", false, "2", "High", projectMMS, userAssignee, parent77913, "Sprint 44 - Fortune Squad"),
		createIssue("10502", "QR-202", "Query Review: analyze partition pruning on daily settlement logs", "Verify partition pruning works deterministically on monthly table partitions.", "query", 2, "Query Review", "1", "To Do", "blue-gray", "Task", false, "3", "Medium", projectMMS, userAssignee, parent77913, "Sprint 45 - Fortune Squad"),
		createIssue("10503", "QR-203", "Query Review: compound index tuning for merchant search by NIK/NPWP", "Add compound partial index to reduce heap fetch latency.", "query", 2, "Query Review", "5", "Done", "green", "Task", false, "3", "Medium", projectMMS, userAssignee, parent77911, "Sprint 44 - Fortune Squad"),
		createIssue("10601", "SOP-301", "SOP MMS Pre-Deployment Checklist & Database Backup Verification", "Pre-deployment verification procedure and rollback validation checklist.", "sop", 1, "SOP MMS", "5", "Done", "green", "Task", false, "3", "Medium", projectMMS, userAssignee, nil, "Sprint 44 - Fortune Squad"),
		createIssue("10602", "SOP-302", "SOP Secret Rotation Procedure for Atlassian & DeepSeek API Keys", "Quarterly key rotation runbook for Vault and encrypted application secrets.", "sop", 1, "SOP MMS", "5", "Done", "green", "Task", false, "3", "Medium", projectMMS, userAssignee, nil, "Sprint 44 - Fortune Squad"),
		createIssue("10603", "SOP-303", "SOP Emergency Rollback Procedure for Settlement Batch Daemon", "Step-by-step emergency drain and rollback runbook for settlement consumers.", "sop", 1, "SOP MMS", "1", "To Do", "blue-gray", "Task", false, "2", "High", projectMMS, userAssignee, nil, "Sprint 45 - Fortune Squad"),
		createIssue("10604", "SOP-304", "SOP ServiceMap Knowledge Base Synchronization and Re-indexing", "Runbook for weekly ServiceMap sqlite FTS5 schema rebuild and verification.", "sop", 1, "SOP MMS", "3", "In Progress", "yellow", "Task", false, "3", "Medium", projectMMS, userAssignee, nil, "Sprint 44 - Fortune Squad"),
		createIssue("10605", "SOP-305", "SOP Post-Deployment Smoke Test Execution on Staging Environment", "Automated smoke tests executing API checks across all core endpoints.", "sop", 1, "SOP MMS", "1", "To Do", "blue-gray", "Task", false, "3", "Medium", projectMMS, userAssignee, nil, "Sprint 45 - Fortune Squad"),
		createIssue("10606", "SOP-306", "SOP Disaster Recovery Failover Testing for Redis Cluster", "Bi-annual failover test validating high availability sentinel election.", "sop", 2, "SOP MMS", "1", "To Do", "blue-gray", "Task", false, "2", "High", projectMMS, userAssignee, nil, "Sprint 45 - Fortune Squad"),
		createIssue("10701", "AZZ-101", "Fix reconciliation timeout in settlement pipeline", "Address gateway latency spike causing reconciliation timeout.", "bug", 3, "Bug Fix", "3", "In Progress", "yellow", "Bug", false, "2", "High", projectMMS, nil, nil, "Sprint Azzuri #5"),
		createIssue("10702", "AZZ-102", "Sync merchant terminal status with core bank ledger", "Batch cron sync for terminal heartbeat status.", "task", 5, "MMS Core", "1", "To Do", "blue-gray", "Task", false, "3", "Medium", projectMMS, nil, nil, "Sprint Azzuri #5"),
	}
}

func createIssue(
	id string,
	key string,
	summary string,
	description string,
	kind string,
	points int,
	subLabel string,
	statusID string,
	statusName string,
	statusColor string,
	typeName string,
	isSubtask bool,
	priorityID string,
	priorityName string,
	project domain.JiraProject,
	assignee *domain.JiraUser,
	parent *domain.JiraParent,
	sprintName string,
) domain.JiraIssue {
	return domain.JiraIssue{
		ID:         id,
		Key:        key,
		Self:       "https://jira.bri.co.id/rest/api/2/issue/" + id,
		SprintName: sprintName,
		Kind:       kind,
		Points:     points,
		SubLabel:   subLabel,
		Fields: domain.JiraIssueFields{
			Summary:     summary,
			Description: description,
			Updated:     "2026-09-29T10:15:00.000+0700",
			Created:     "2026-09-20T08:00:00.000+0700",
			Project:     project,
			Status: domain.JiraStatus{
				ID:          statusID,
				Name:        statusName,
				Description: statusName + " state",
				StatusCategory: domain.JiraStatusCategory{
					ID:        1,
					Key:       statusName,
					Name:      statusName,
					ColorName: statusColor,
				},
			},
			IssueType: &domain.JiraIssueType{
				ID:      typeName,
				Name:    typeName,
				Subtask: isSubtask,
			},
			Priority: &domain.JiraPriority{
				ID:   priorityID,
				Name: priorityName,
			},
			Assignee: assignee,
			Parent:   parent,
		},
	}
}

func buildSeedSprints(issues []domain.JiraIssue) []domain.JiraSprint {
	sprint44Issues := make([]domain.JiraIssue, 0)
	sprint45Issues := make([]domain.JiraIssue, 0)
	sprintAzzuriIssues := make([]domain.JiraIssue, 0)

	for _, issue := range issues {
		if issue.SprintName == "Sprint 44 - Fortune Squad" {
			sprint44Issues = append(sprint44Issues, issue)
		} else if issue.SprintName == "Sprint 45 - Fortune Squad" {
			sprint45Issues = append(sprint45Issues, issue)
		} else if issue.SprintName == "Sprint Azzuri #5" {
			sprintAzzuriIssues = append(sprintAzzuriIssues, issue)
		}
	}

	return []domain.JiraSprint{
		{
			ID:            44,
			Name:          "Sprint 44 - Fortune Squad",
			State:         "active",
			StartDate:     "2026-09-15T00:00:00.000Z",
			EndDate:       "2026-10-02T23:59:59.000Z",
			OriginBoardID: 2646,
			Issues:        sprint44Issues,
		},
		{
			ID:            45,
			Name:          "Sprint 45 - Fortune Squad",
			State:         "future",
			StartDate:     "2026-10-03T00:00:00.000Z",
			EndDate:       "2026-10-20T23:59:59.000Z",
			OriginBoardID: 2646,
			Issues:        sprint45Issues,
		},
		{
			ID:            55,
			Name:          "Sprint Azzuri #5",
			State:         "active",
			StartDate:     "2026-09-18T00:00:00.000Z",
			EndDate:       "2026-10-05T23:59:59.000Z",
			OriginBoardID: 2646,
			Issues:        sprintAzzuriIssues,
		},
	}
}

var seedConfluencePageIDs = map[string][]string{
	"CRMMS-77911": {"110001"},
	"CRMMS-77913": {"110002", "110003"},
	"CRMMS-77914": {"110004"},
	"CRMMS-77925": {"110005"},
}

func buildSeedRemoteLinks(issueKey string) []domain.JiraRemoteLink {
	pageIDs, exists := seedConfluencePageIDs[strings.ToUpper(strings.TrimSpace(issueKey))]
	if !exists {
		return []domain.JiraRemoteLink{}
	}

	links := make([]domain.JiraRemoteLink, 0, len(pageIDs))
	for index, pageID := range pageIDs {
		links = append(links, domain.JiraRemoteLink{
			ID:           7000 + index,
			Relationship: "wiki",
			Object: domain.JiraRemoteLinkObject{
				URL:   "https://confluence.bri.co.id/pages/viewpage.action?pageId=" + pageID,
				Title: "Wiki Page " + pageID,
			},
		})
	}
	return links
}
