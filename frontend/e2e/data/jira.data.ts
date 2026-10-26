export const mockAssignedIssues = [
  {
    id: "UT-101",
    key: "UT-101",
    kind: "ut",
    label: "UT",
    title: "Auth middleware unit tests",
    status: "open",
    priority: "low",
    points: 3,
    sub: "Auth · v2 middleware",
    description: "Unit test suite covering token parsing, refresh handling, and error paths for the v2 auth middleware."
  },
  {
    id: "UT-102",
    key: "UT-102",
    kind: "ut",
    label: "UT",
    title: "Payment gateway integration tests",
    status: "open",
    priority: "medium",
    points: 5,
    sub: "Payments · sandbox",
    description: "Integration tests against the sandbox payment gateway, covering successful charges and declines."
  },
  {
    id: "UT-103",
    key: "UT-103",
    kind: "ut",
    label: "UT",
    title: "Notification service tests",
    status: "open",
    priority: "low",
    points: 2,
    sub: "Notifications · templating",
    description: "Tests for notification templating, deduplication, and delivery retries."
  },
  {
    id: "QR-201",
    key: "QR-201",
    kind: "query",
    label: "QR",
    title: "Monthly transaction reconciliation query",
    status: "open",
    priority: "medium",
    points: 2,
    sub: "Reconciliation · monthly close",
    description: "SQL query used to reconcile transaction records against the ledger for monthly close."
  },
  {
    id: "QR-202",
    key: "QR-202",
    kind: "query",
    label: "QR",
    title: "User activity audit query",
    status: "open",
    priority: "high",
    points: 3,
    sub: "Audit · compliance",
    description: "Audit query listing user activity over the last 30 days for compliance verification."
  },
  {
    id: "SOP-301",
    key: "SOP-301",
    kind: "sop",
    label: "SOP",
    title: "Monitoring — Daily health check",
    status: "open",
    priority: "medium",
    points: 1,
    sub: "Monitoring · daily",
    description: "Step-by-step procedure for the daily health check of production services."
  },
  {
    id: "SOP-302",
    key: "SOP-302",
    kind: "sop",
    label: "SOP",
    title: "Monitoring — Alert escalation matrix",
    status: "open",
    priority: "high",
    points: 2,
    sub: "Monitoring · escalation",
    description: "Escalation matrix and paging rules for production alerts and on-call response."
  },
  {
    id: "CRMMS-77911",
    key: "CRMMS-77911",
    kind: "ut",
    label: "UT",
    title: "Sebagai Administrator, I want to melihat daftar user yang dikelompokkan berdasarkan Personal Number (PN), mengelola level/jabatan mereka dalam satu halaman detail, serta melihat riwayat perubahannya",
    status: "progress",
    priority: "high",
    points: 5,
    sub: "Core · user management",
    description: "Sebagai Administrator sistem MMS BRI, saya membutuhkan halaman komprehensif untuk memantau data seluruh user berdasarkan PN mereka. Detail mencakup pengelolaan hak akses per modul, histori login dan aktivitas 30 hari terakhir, serta approval workflow untuk perubahan level/jabatan secara terpusat.",
    generated: "describe('CRMMS-77911 user management', () => {\n  it('groups users by personal number', () => {});\n  it('updates role and permissions atomically', () => {});\n});"
  },
  {
    id: "QR-203",
    key: "QR-203",
    kind: "query",
    label: "QR",
    title: "Duplicate transaction detection query",
    status: "progress",
    priority: "medium",
    points: 2,
    sub: "Ops · daily cleanup",
    description: "Query detecting duplicate transactions by hash within a 24-hour window."
  },
  {
    id: "SOP-303",
    key: "SOP-303",
    kind: "sop",
    label: "SOP",
    title: "Operational — Deploy checklist",
    status: "done",
    priority: "low",
    points: 1,
    sub: "Operational · deploy",
    description: "Pre- and post-deploy checklist covering migrations, smoke tests, and rollback triggers."
  },
  {
    id: "BUG-201",
    key: "BUG-201",
    kind: "bug",
    label: "BUG",
    title: "Checkout fails when cart has an expired item",
    status: "open",
    priority: "high",
    points: 3,
    sub: "QA · Sinta · 2h ago",
    description: "Checkout flow fails abruptly when one of the cart items has passed its expiration timestamp."
  },
  {
    id: "BUG-202",
    key: "BUG-202",
    kind: "bug",
    label: "BUG",
    title: "Session timeout does not refresh the auth token",
    status: "progress",
    priority: "high",
    points: 3,
    sub: "QA · Dimas · 4h ago",
    description: "When user session times out after 15 minutes of inactivity, API requests fail with 401 instead of silent refresh."
  },
  {
    id: "SUB-101",
    key: "SUB-101",
    kind: "subtask",
    label: "SUB",
    title: "Write unit tests for auth middleware",
    status: "open",
    priority: "medium",
    points: 2,
    sub: "Parent · LUN-501",
    description: "Add comprehensive unit test coverage for token verification and claims validation."
  },
  {
    id: "SUB-102",
    key: "SUB-102",
    kind: "subtask",
    label: "SUB",
    title: "Update API docs for v2 endpoints",
    status: "done",
    priority: "low",
    points: 1,
    sub: "Parent · LUN-501",
    description: "Document breaking changes and new query parameters in REST API v2."
  }
]

export const mockAssignedIssuesWithoutSop = mockAssignedIssues.filter(
  (issue) => issue.kind !== "sop"
)

export const mockBacklogResponse = {
  sprints: [
    {
      id: "sprint-44",
      name: "Sprint 44 - Fortune Squad",
      is_active: true,
      dates: "Sep 15 - Sep 29",
      issues: [
        {
          id: "CRMMS-77911",
          key: "CRMMS-77911",
          kind: "story",
          title: "Sebagai Administrator, I want to melihat daftar user yang dikelompokkan berdasarkan Personal Number (PN), mengelola level/jabatan mereka dalam satu halaman detail, serta melihat riwayat perubahannya",
          status: "progress",
          priority: "high",
          points: 5,
          description: "Sebagai Administrator sistem MMS BRI, saya membutuhkan halaman komprehensif untuk memantau data seluruh user berdasarkan PN mereka."
        }
      ]
    },
    {
      id: "sprint-45",
      name: "Sprint 45 - Next Phase",
      is_active: false,
      dates: "Sep 30 - Oct 14",
      issues: [
        {
          id: "CRMMS-8812",
          key: "CRMMS-8812",
          kind: "task",
          title: "Merchant transaction settlement batch job",
          status: "open",
          priority: "medium",
          points: 3,
          description: "Daily automated settlement batch processing for registered MMS merchants."
        }
      ]
    }
  ],
  total: 2
}
