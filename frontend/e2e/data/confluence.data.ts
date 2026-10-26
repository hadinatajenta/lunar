export const mockConfluenceDocuments = [
  {
    id: "UT-101",
    type: "ut",
    type_label: "UT",
    title: "UT-101 User Authentication Flow",
    status: "open",
    owner: "Eren",
    updated: "2026-09-28T10:00:00.000+07:00",
    space: "Engineering / Auth",
    description: "Unit test specification for the new authentication flow covering login, token refresh, and session invalidation.",
    url: "https://confluence.bri.co.id/pages/viewpage.action?pageId=UT-101"
  },
  {
    id: "QR-201",
    type: "query",
    type_label: "QR",
    title: "QR-201 Slow Query Review",
    status: "progress",
    owner: "Siti",
    updated: "2026-09-27T14:30:00.000+07:00",
    space: "DBA / Reviews",
    description: "Query review for CRMMS-77911: index missing on users.personal_number causing sequential scan on 2.4M rows.",
    url: "https://confluence.bri.co.id/pages/viewpage.action?pageId=QR-201"
  },
  {
    id: "SOP-301",
    type: "sop",
    type_label: "SOP",
    title: "SOP-301 Database Migration",
    status: "done",
    owner: "Budi",
    updated: "2026-09-26T09:15:00.000+07:00",
    space: "DBA / SOPs",
    description: "Standard operating procedure for zero-downtime schema migrations with backward-compatible rollout.",
    url: "https://confluence.bri.co.id/pages/viewpage.action?pageId=SOP-301"
  },
  {
    id: "UT-102",
    type: "ut",
    type_label: "UT",
    title: "UT-102 Payment Idempotency",
    status: "open",
    owner: "Eren",
    updated: "2026-09-25T16:45:00.000+07:00",
    space: "Engineering / Payments",
    description: "Idempotency key design and test matrix for payment initiation endpoints.",
    url: "https://confluence.bri.co.id/pages/viewpage.action?pageId=UT-102"
  },
  {
    id: "QR-202",
    type: "query",
    type_label: "QR",
    title: "QR-202 Index Suggestion",
    status: "open",
    owner: "Siti",
    updated: "2026-09-24T11:20:00.000+07:00",
    space: "DBA / Reviews",
    description: "Composite index on orders(created_at, user_id) to support pagination query in CRMMS-78113.",
    url: "https://confluence.bri.co.id/pages/viewpage.action?pageId=QR-202"
  },
  {
    id: "DOC-401",
    type: "doc",
    type_label: "DOC",
    title: "DOC-401 Architecture Overview",
    status: "open",
    owner: "Eren",
    updated: "2026-09-23T08:00:00.000+07:00",
    space: "Engineering / Architecture",
    description: "High-level system architecture diagram and component interaction flow for the Lunar workspace.",
    url: ""
  }
]

export const mockQueryReviewResult = {
  document_id: "QR-201",
  model: "DeepSeek-V4 Pro (Thinking)",
  prompt_version: "queryreview-v1",
  mig_page_id: "2086971956",
  mig_page_title: "CRMMS-76972-MIG",
  skipped_repositories: [{ repository: "fe-mms", reason: "excluded repository" }],
  repositories: [
    {
      repository: "micro-payment",
      project_key: "NME",
      source_branch: "feat/CRMMS-123",
      target_branch: "development",
      compare_url:
        "https://bitbucket.bri.co.id/projects/NME/repos/micro-payment/compare/diff?sourceBranch=feat/CRMMS-123&targetBranch=development",
      status: "success",
      analyzed_files: [
        { path: "internal/repo/dao.go", added_lines: 12, language: "go" },
        { path: "internal/repo/user.go", added_lines: 4, language: "go" }
      ],
      skipped_files: [{ path: "internal/repo/dao_test.go", reason: "test file" }],
      queries: [
        {
          no: 1,
          feature: "[micro-payment] Get user by PN",
          repository: "micro-payment",
          table: "user_data",
          action_type: "SELECT",
          query_statement: "select id, username from user_data where pn = '[input_here]'",
          index: "pn",
          create_index: "CREATE INDEX idx_user_data_pn ON user_data (pn);",
          file: "internal/repo/dao.go",
          line: 42,
          notes: "Filtered by personal number."
        }
      ],
      summary: "One SELECT was introduced on this branch.",
      truncated: false,
      error: ""
    },
    {
      repository: "micro-notification",
      project_key: "NME",
      source_branch: "feat/CRMMS-123",
      target_branch: "development",
      compare_url:
        "https://bitbucket.bri.co.id/projects/NME/repos/micro-notification/compare/diff?sourceBranch=feat/CRMMS-123&targetBranch=development",
      status: "success",
      analyzed_files: [{ path: "internal/repo/audit.go", added_lines: 6, language: "go" }],
      skipped_files: [],
      queries: [
        {
          no: 1,
          feature: "[micro-notification] Insert audit log",
          repository: "micro-notification",
          table: "audit_log",
          action_type: "INSERT",
          query_statement: "insert into audit_log (message) values ('[input_here]')",
          index: "PRIMARY",
          create_index: "",
          file: "internal/repo/audit.go",
          line: 18,
          notes: ""
        }
      ],
      summary: "One INSERT was introduced on this branch.",
      truncated: false,
      error: ""
    }
  ]
}

export const mockQueryReviewNoQueries = {
  document_id: "QR-201",
  model: "DeepSeek-V4 Pro (Thinking)",
  prompt_version: "queryreview-v1",
  mig_page_id: "2086971956",
  mig_page_title: "CRMMS-76972-MIG",
  skipped_repositories: [{ repository: "fe-mms", reason: "excluded repository" }],
  repositories: [
    {
      repository: "micro-payment",
      project_key: "NME",
      source_branch: "feat/CRMMS-123",
      target_branch: "development",
      compare_url:
        "https://bitbucket.bri.co.id/projects/NME/repos/micro-payment/compare/diff?sourceBranch=feat/CRMMS-123&targetBranch=development",
      status: "no_queries",
      analyzed_files: [{ path: "internal/repo/service.go", added_lines: 3, language: "go" }],
      skipped_files: [],
      queries: [],
      summary: "Only service wiring changed on this branch.",
      truncated: false,
      error: ""
    }
  ]
}

export const mockQueryReviewMixed = {
  document_id: "QR-201",
  model: "DeepSeek-V4 Pro (Thinking)",
  prompt_version: "queryreview-v1",
  mig_page_id: "2086971956",
  mig_page_title: "CRMMS-76972-MIG",
  skipped_repositories: [{ repository: "fe-mms", reason: "excluded repository" }],
  repositories: [
    {
      repository: "micro-payment",
      project_key: "NME",
      source_branch: "feat/CRMMS-123",
      target_branch: "development",
      compare_url:
        "https://bitbucket.bri.co.id/projects/NME/repos/micro-payment/compare/diff?sourceBranch=feat/CRMMS-123&targetBranch=development",
      status: "success",
      analyzed_files: [{ path: "internal/repo/dao.go", added_lines: 12, language: "go" }],
      skipped_files: [],
      queries: [
        {
          no: 1,
          feature: "[micro-payment] Get user by PN",
          repository: "micro-payment",
          table: "user_data",
          action_type: "SELECT",
          query_statement: "select id, username from user_data where pn = '[input_here]'",
          index: "pn",
          create_index: "CREATE INDEX idx_user_data_pn ON user_data (pn);",
          file: "internal/repo/dao.go",
          line: 42,
          notes: ""
        }
      ],
      summary: "One SELECT was introduced on this branch.",
      truncated: false,
      error: ""
    },
    {
      repository: "micro-notification",
      project_key: "NME",
      source_branch: "feat/CRMMS-123",
      target_branch: "development",
      compare_url:
        "https://bitbucket.bri.co.id/projects/NME/repos/micro-notification/compare/diff?sourceBranch=feat/CRMMS-123&targetBranch=development",
      status: "error",
      analyzed_files: [],
      skipped_files: [],
      queries: [],
      summary: "",
      truncated: false,
      error: "The compare diff could not be fetched: upstream returned 502."
    }
  ]
}

export const mockQueryReviewNoCompareLink = {
  code: "NO_COMPARE_LINK",
  message: "The document does not contain a Bitbucket compare link."
}

export const mockQueryReviewApplyResult = {
  page_id: "2144721484",
  version: 12,
  url: "https://confluence.bri.co.id/pages/viewpage.action?pageId=2144721484"
}

export const mockQueryReviewApplyConflict = {
  message: "The page was updated elsewhere. Reload and try again."
}

export const mockQueryReviewEmpty = {
  document_id: "QR-201",
  model: "DeepSeek-V4 Pro (Thinking)",
  prompt_version: "queryreview-v1",
  mig_page_id: "",
  mig_page_title: "",
  skipped_repositories: [],
  repositories: []
}

export const mockUtRunDone = {
  run_id: "run-e2e-1",
  repo: "micro-payment",
  branch: "feat/CRMMS-76972",
  base_ref: "development",
  changed_files: ["internal/repo/dao.go"],
  test_files: [
    {
      path: "internal/repo/dao_test.go",
      package: "./internal/repo",
      language: "go",
      classification: "safe",
      reason: ""
    }
  ],
  blocked_files: [
    {
      path: "internal/seed/reset_test.go",
      package: "./internal/seed",
      language: "go",
      classification: "blocked",
      reason: "calls OpenDB and TRUNCATE"
    }
  ],
  run: {
    status: "passed",
    executed_packages: ["./internal/repo"],
    total: 2,
    passed: 2,
    failed: 0,
    duration_ms: 4200,
    output_excerpt: "ok ./internal/repo"
  },
  stashed: true,
  restored: true,
  tests: [
    {
      name: "TestCreatePayment",
      package: "./internal/repo",
      file: "internal/repo/dao_test.go",
      status: "passed",
      duration_ms: 1200,
      output: "--- PASS: TestCreatePayment"
    },
    {
      name: "TestUpdatePayment",
      package: "./internal/repo",
      file: "internal/repo/dao_test.go",
      status: "failed",
      duration_ms: 900,
      output: "--- FAIL: TestUpdatePayment"
    }
  ],
  untested_functions: [
    {
      name: "RefundPayment",
      file: "internal/repo/refund.go",
      package: "./internal/repo"
    }
  ],
  screenshot_url: "/ut-runs/run-e2e-1/screenshot.png"
}

export const buildMockUtRunDone = (repository: string) => ({
  ...mockUtRunDone,
  run_id: `run-${repository}`,
  repo: repository,
  screenshot_url: `/ut-runs/run-${repository}/screenshot.png`
})

export const mockMigTargets = {
  mig_page_title: "CRMMS-76972-MIG",
  targets: [
    { repository: "micro-payment", branch: "feat/CRMMS-76972" },
    { repository: "micro-notification", branch: "feat/CRMMS-76972" }
  ],
  skipped_repositories: [
    { repository: "fe-mms", reason: "excluded repository" },
    { repository: "mms", reason: "excluded repository" }
  ]
}

export const mockMigTargetsSingle = {
  mig_page_title: "CRMMS-76972-MIG",
  targets: [{ repository: "micro-payment", branch: "feat/CRMMS-76972" }],
  skipped_repositories: [{ repository: "fe-mms", reason: "excluded repository" }]
}

export const mockMigTargetsEmpty = {
  mig_page_title: "CRMMS-76972-MIG",
  targets: [],
  skipped_repositories: []
}

export const mockUtDocsResult = {
  document_id: "UT-101",
  model: "DeepSeek-V4 Pro (Thinking)",
  prompt_version: "utdocs-v1",
  title: "UT documentation for micro-payment",
  summary: "Two mock-only tests ran for micro-payment and one file was blocked before execution.",
  table: "<table></table>",
  rows: [
    {
      no: 1,
      repository: "micro-payment",
      function: "CreatePayment",
      scenario: "Creates a payment with a valid payload",
      expected_result: "The payment is stored",
      input_data: "valid payload",
      result: "Passed",
      screen_capture: "run-micro-payment.png",
      test_function_name: "TestCreatePayment"
    },
    {
      no: 2,
      repository: "micro-payment",
      function: "UpdatePayment",
      scenario: "Updates an existing payment",
      expected_result: "The payment is updated",
      input_data: "changed amount",
      result: "Failed",
      screen_capture: "run-micro-payment.png",
      test_function_name: "TestUpdatePayment"
    }
  ],
  markdown: "# UT documentation for micro-payment",
  blocked_notice: "1 test file was not run because it can reset a database",
  screenshot_caption: "UT run for micro-payment on feat/CRMMS-76972",
  mock_tests: []
}

export const mockUtDocsResultGrouped = {
  document_id: "UT-101",
  model: "DeepSeek-V4 Pro (Thinking)",
  prompt_version: "utdocs-v1",
  title: "UT documentation for the MIG page",
  summary: "Two repositories ran their mock-only unit tests.",
  table: "<table></table>",
  rows: [
    {
      no: 1,
      repository: "micro-payment",
      function: "CreatePayment",
      scenario: "Creates a payment with a valid payload",
      expected_result: "The payment is stored",
      input_data: "valid payload",
      result: "Passed",
      screen_capture: "run-micro-payment.png",
      test_function_name: "TestCreatePayment"
    },
    {
      no: 2,
      repository: "micro-payment",
      function: "UpdatePayment",
      scenario: "Updates an existing payment",
      expected_result: "The payment is updated",
      input_data: "changed amount",
      result: "Failed",
      screen_capture: "run-micro-payment.png",
      test_function_name: "TestUpdatePayment"
    },
    {
      no: 1,
      repository: "micro-notification",
      function: "SendAudit",
      scenario: "Sends an audit entry",
      expected_result: "The entry is queued",
      input_data: "audit payload",
      result: "Passed",
      screen_capture: "run-micro-notification.png",
      test_function_name: "TestSendAudit"
    },
    {
      no: 2,
      repository: "micro-notification",
      function: "PurgeAudit",
      scenario: "Purges old audit entries",
      expected_result: "Old entries are removed",
      input_data: "retention window",
      result: "Skipped",
      screen_capture: "run-micro-notification.png",
      test_function_name: "TestPurgeAudit"
    }
  ],
  markdown: "# UT documentation for the MIG page",
  blocked_notice: "1 test file was not run because it can reset a database",
  screenshot_caption: "UT run for the MIG page",
  mock_tests: []
}

export const MOCK_REFUND_TEST_CODE = [
  "func TestRefundPaymentMock(t *testing.T) {",
  "\trepo := newMockRepository()",
  '\tresult, err := RefundPayment(repo, "pay-1")',
  "\tif err != nil {",
  '\t\tt.Fatalf("unexpected error: %v", err)',
  "\t}",
  '\tif result.Status != "refunded" {',
  '\t\tt.Fatalf("want refunded, got %s", result.Status)',
  "\t}",
  "}"
].join("\n")

export const mockUtDocsResultWithMockCode = {
  document_id: "UT-101",
  model: "DeepSeek-V4 Pro (Thinking)",
  prompt_version: "utdocs-v1",
  title: "UT documentation for micro-payment",
  summary: "One mock-only test ran and one changed function has a mock test draft.",
  table: "<table></table>",
  rows: [
    {
      no: 1,
      repository: "micro-payment",
      function: "CreatePayment",
      scenario: "Creates a payment with a valid payload",
      expected_result: "The payment is stored",
      input_data: "valid payload",
      result: "Passed",
      screen_capture: "run-micro-payment.png",
      test_function_name: "TestCreatePayment"
    },
    {
      no: 2,
      repository: "micro-payment",
      function: "RefundPayment",
      scenario: "Refunds a captured payment",
      expected_result: "The payment is refunded",
      input_data: "payment id",
      result: "Not Run",
      screen_capture: "",
      test_function_name: "TestRefundPaymentMock"
    }
  ],
  mock_tests: [
    {
      test_function_name: "TestRefundPaymentMock",
      function: "RefundPayment",
      scenario: "Refunds a captured payment",
      expected_result: "The payment is refunded",
      input_data: "payment id",
      code: MOCK_REFUND_TEST_CODE
    }
  ],
  markdown: "# UT documentation for micro-payment",
  blocked_notice: "",
  screenshot_caption: "UT run for micro-payment on feat/CRMMS-76972"
}

export const mockUtDocsResultNotRunOnly = {
  document_id: "UT-101",
  model: "DeepSeek-V4 Pro (Thinking)",
  prompt_version: "utdocs-v1",
  title: "UT documentation for micro-payment",
  summary: "Every changed function still needs a mock test.",
  table: "<table></table>",
  rows: [
    {
      no: 1,
      repository: "micro-payment",
      function: "RefundPayment",
      scenario: "Refunds a captured payment",
      expected_result: "The payment is refunded",
      input_data: "payment id",
      result: "Not Run",
      screen_capture: "",
      test_function_name: "TestRefundPaymentMock"
    }
  ],
  mock_tests: [
    {
      test_function_name: "TestRefundPaymentMock",
      function: "RefundPayment",
      scenario: "Refunds a captured payment",
      expected_result: "The payment is refunded",
      input_data: "payment id",
      code: MOCK_REFUND_TEST_CODE
    }
  ],
  markdown: "# UT documentation for micro-payment",
  blocked_notice: "",
  screenshot_caption: "UT run for micro-payment on feat/CRMMS-76972"
}

export const mockUtDocsApplyResult = {
  page_id: "2144721484",
  version: 13,
  url: "https://confluence.bri.co.id/pages/viewpage.action?pageId=2144721484"
}

export const mockUtDocsApplyConflict = {
  message: "The page was updated elsewhere. Reload and try again."
}

export const mockSopDocument = {
  id: "2144721483",
  type: "sop",
  type_label: "SOP",
  title: "CRMMS-78064-[2026MMDD] Dokumen Query Review",
  status: "progress",
  owner: "Siti",
  updated: "2026-09-27T14:30:00.000+07:00",
  space: "DBA / Reviews",
  description: "Wiki page opened from CRMMS-78064.",
  url: "https://confluence.bri.co.id/pages/viewpage.action?pageId=2144721483"
}

export const mockSopDocumentWithoutIssueKey = {
  ...mockSopDocument,
  id: "2144721999",
  title: "Architecture Overview",
  description: "A wiki page without an issue key in its title."
}

export const mockSopGenerateResult = {
  issue_key: "CRMMS-78064",
  development_page_id: "2144721483",
  repositories: [
    {
      repository: "everest",
      branch: "feature/CRMMS-77914",
      base_ref: "master",
      database_files: [],
      warnings: []
    },
    {
      repository: "fe-mms",
      branch: "feature/CRMMS-77914",
      database_files: [],
      warnings: []
    }
  ],
  deploy: {
    title: "CRMMS-78064-SOP Deploy",
    page_id: "2144721548",
    exists: true,
    managed_block:
      "<p><strong>BAB I. DEPLOY DATABASE MMS</strong></p><p>Tidak ada penambahan Table/Database</p><p><strong>BAB II. DEPLOY APLIKASI</strong></p><ol><li>Build everest on release/production</li><li>Deploy to the DC environment</li></ol><p><strong>BAB III. ROLLBACK APLIKASI</strong></p><ol><li>Select the previous Bamboo plan</li></ol>"
  },
  operasional: {
    title: "CRMMS-78064-SOP Operasional",
    page_id: "2144721557",
    exists: true,
    children: [
      { title: "Maintenance", exists: true },
      { title: "Monitoring", exists: false },
      { title: "Troubleshooting", exists: false }
    ]
  },
  warnings: ["no Bamboo plan configured for everest"]
}

export const mockSopGenerateResultNewPage = {
  ...mockSopGenerateResult,
  deploy: {
    title: "CRMMS-78064-SOP Deploy",
    page_id: "",
    exists: false,
    managed_block: mockSopGenerateResult.deploy.managed_block
  }
}

export const mockSopGenerateResultScanned = {
  ...mockSopGenerateResult,
  repositories: [
    {
      repository: "everest",
      branch: "feature/CRMMS-77914",
      base_ref: "master",
      database_files: [
        {
          path: "migrations/2026_add_mdr.sql",
          statements: [
            {
              action: "CREATE",
              table: "log_program_mdr",
              sql: "CREATE TABLE log_program_mdr (id bigint primary key)"
            }
          ]
        }
      ],
      warnings: []
    },
    {
      repository: "fe-mms",
      branch: "feature/CRMMS-77914",
      database_files: [],
      warnings: []
    }
  ],
  deploy: {
    title: "CRMMS-78064-SOP Deploy",
    page_id: "2144721548",
    exists: true,
    managed_block:
      "<p><strong>BAB I. DEPLOY DATABASE MMS</strong></p><table><tbody><tr><td>log_program_mdr</td><td>CREATE</td></tr></tbody></table><p><strong>BAB II. DEPLOY APLIKASI</strong></p><ol><li>Build everest on release/production</li><li>Deploy to the DC environment</li></ol><p><strong>BAB III. ROLLBACK APLIKASI</strong></p><ol><li>Select the previous Bamboo plan</li></ol>"
  }
}

export const buildMockSopScanResult = (repository: string) => ({
  repository,
  branch: "feature/CRMMS-77914",
  base_ref: "master",
  database_files:
    repository === "everest"
      ? [
          {
            path: "migrations/2026_add_mdr.sql",
            statements: [
              {
                action: "CREATE",
                table: "log_program_mdr",
                sql: "CREATE TABLE log_program_mdr (id bigint primary key)"
              }
            ]
          }
        ]
      : [],
  tables: repository === "everest" ? [{ name: "log_program_mdr", action: "CREATE" }] : [],
  warnings: []
})

export const mockSopApplyResult = {
  deploy_page: {
    id: "2144721548",
    version: 14,
    url: "https://confluence.bri.co.id/pages/viewpage.action?pageId=2144721548",
    action: "updated"
  },
  operasional_page: {
    id: "2144721557",
    version: 3,
    url: "https://confluence.bri.co.id/pages/viewpage.action?pageId=2144721557"
  },
  created_children: ["Monitoring", "Troubleshooting"]
}

export const mockSopApplyConflict = {
  message: "The page was updated elsewhere. Reload and try again."
}
