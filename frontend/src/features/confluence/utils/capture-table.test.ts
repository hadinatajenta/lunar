import assert from "node:assert/strict"
import { describe, test } from "node:test"
import type { QueryReviewEntry, QueryReviewRepository } from "../types.ts"
import { buildCaptureTableHtml, buildCaptureTableText } from "./capture-table.ts"

const buildEntry = (overrides: Partial<QueryReviewEntry> = {}): QueryReviewEntry => ({
  no: 1,
  feature: "[micro-payment] Get user by PN",
  repository: "micro-payment",
  table: "user_data",
  action_type: "SELECT",
  query_statement: "select id, username from user_data where pn = ?",
  index: "pn",
  create_index: "",
  file: "internal/repo/dao.go",
  line: 42,
  notes: "",
  ...overrides
})

const buildRepository = (
  overrides: Partial<QueryReviewRepository> = {}
): QueryReviewRepository => ({
  repository: "micro-payment",
  project_key: "NME",
  source_branch: "feat/CRMMS-123",
  target_branch: "development",
  compare_url: "https://bitbucket.bri.co.id/projects/NME/repos/micro-payment/compare/diff",
  status: "success",
  analyzed_files: [],
  skipped_files: [],
  queries: [buildEntry()],
  summary: "",
  truncated: false,
  error: "",
  ...overrides
})

describe("buildCaptureTableHtml", () => {
  test("renders the five headers and an empty last column for a single repository", () => {
    const html = buildCaptureTableHtml([buildRepository()])

    assert.match(html, /^<table class="wrapped relative-table" style="width: 100%;">/)
    assert.match(html, /<th>No<\/th>/)
    assert.match(html, /<th>Fitur<\/th>/)
    assert.match(html, /<th>Query Statement<\/th>/)
    assert.match(html, /<th>Index<\/th>/)
    assert.match(html, /<th>Capture Explain \/ Execution Plan<\/th>/)
    assert.match(html, /<td>\[micro-payment\] Get user by PN<\/td>/)
    assert.match(html, /<td>pn<\/td><td><\/td>/)
    assert.doesNotMatch(html, /Layanan \/ Microservice/)
  })

  test("renders the possible key and the CREATE INDEX statement in the Index cell", () => {
    const html = buildCaptureTableHtml([
      buildRepository({
        queries: [
          buildEntry({
            index: "pn",
            create_index: "CREATE INDEX idx_user_data_pn ON user_data (pn);"
          })
        ]
      })
    ])

    assert.match(html, /<td>pn<br>CREATE INDEX idx_user_data_pn ON user_data \(pn\);<\/td>/)
  })

  test("renders only the possible key when no CREATE INDEX is suggested", () => {
    const html = buildCaptureTableHtml([buildRepository()])

    assert.match(html, /<td>pn<\/td>/)
    assert.doesNotMatch(html, /CREATE INDEX/)
  })

  test("escapes the CREATE INDEX statement so it cannot inject markup", () => {
    const html = buildCaptureTableHtml([
      buildRepository({
        queries: [
          buildEntry({
            index: "pn",
            create_index: 'CREATE INDEX "idx" ON t (a<b);'
          })
        ]
      })
    ])

    assert.doesNotMatch(html, /<b>/)
    assert.match(html, /CREATE INDEX &quot;idx&quot; ON t \(a&lt;b\);/)
  })

  test("groups more than one repository and restarts the No column per group", () => {
    const repositories = [
      buildRepository({
        repository: "micro-payment",
        queries: [buildEntry({ no: 1 }), buildEntry({ no: 2, query_statement: "select 2" })]
      }),
      buildRepository({
        repository: "micro-notification",
        queries: [buildEntry({ no: 1, repository: "micro-notification" })]
      })
    ]

    const html = buildCaptureTableHtml(repositories)

    assert.match(
      html,
      /<tr><td colspan="5"><strong>Layanan \/ Microservice: <code>micro-payment<\/code><\/strong><\/td><\/tr>/
    )
    assert.match(
      html,
      /<tr><td colspan="5"><strong>Layanan \/ Microservice: <code>micro-notification<\/code><\/strong><\/td><\/tr>/
    )
    const paymentRows = html.split("micro-notification")[0]
    assert.match(paymentRows, /<tr><td>1<\/td>/)
    assert.match(paymentRows, /<tr><td>2<\/td>/)
    assert.match(html.split("micro-notification")[1], /<tr><td>1<\/td>/)
  })

  test("skips repositories without queries and does not group a lone populated repository", () => {
    const repositories = [
      buildRepository({ repository: "micro-payment", queries: [buildEntry()] }),
      buildRepository({ repository: "micro-report", status: "no_queries", queries: [] })
    ]

    const html = buildCaptureTableHtml(repositories)

    assert.doesNotMatch(html, /Layanan \/ Microservice/)
    assert.doesNotMatch(html, /micro-report/)
  })

  test("escapes every cell value so a statement cannot inject markup", () => {
    const repositories = [
      buildRepository({
        repository: 'pay"<script>',
        queries: [
          buildEntry({
            feature: "<b>bold</b>",
            query_statement: "select * from t where a = '&' and b = \"<script>alert(1)</script>\"",
            index: "idx&on"
          })
        ]
      }),
      buildRepository({ repository: "micro-notification" })
    ]

    const html = buildCaptureTableHtml(repositories)

    assert.doesNotMatch(html, /<script>/)
    assert.doesNotMatch(html, /<b>bold<\/b>/)
    assert.match(html, /&lt;b&gt;bold&lt;\/b&gt;/)
    assert.match(html, /&lt;script&gt;alert\(1\)&lt;\/script&gt;/)
    assert.match(html, /idx&amp;on/)
    assert.match(html, /pay&quot;&lt;script&gt;/)
  })
})

describe("buildCaptureTableText", () => {
  test("writes one line per query with a header line and a group line when grouped", () => {
    const repositories = [
      buildRepository({
        repository: "micro-payment",
        queries: [buildEntry({ no: 1 }), buildEntry({ no: 2, query_statement: "select 2" })]
      }),
      buildRepository({
        repository: "micro-notification",
        queries: [buildEntry({ no: 1, repository: "micro-notification" })]
      })
    ]

    const text = buildCaptureTableText(repositories)
    const lines = text.split("\n")

    assert.equal(
      lines[0],
      "No\tFitur\tQuery Statement\tIndex\tCapture Explain / Execution Plan"
    )
    assert.equal(lines[1], "Layanan / Microservice: micro-payment")
    assert.equal(lines[2], "1\t[micro-payment] Get user by PN\tselect id, username from user_data where pn = ?\tpn\t")
    assert.equal(lines[3], "2\t[micro-payment] Get user by PN\tselect 2\tpn\t")
    assert.equal(lines[4], "Layanan / Microservice: micro-notification")
    assert.equal(lines[5], "1\t[micro-payment] Get user by PN\tselect id, username from user_data where pn = ?\tpn\t")
  })

  test("writes the possible key and the CREATE INDEX statement on separate lines", () => {
    const text = buildCaptureTableText([
      buildRepository({
        queries: [
          buildEntry({
            index: "pn",
            create_index: "CREATE INDEX idx_user_data_pn ON user_data (pn);"
          })
        ]
      })
    ])

    assert.match(
      text,
      /select id, username from user_data where pn = \?\tpn\nCREATE INDEX idx_user_data_pn ON user_data \(pn\);\t/
    )
  })

  test("writes only the possible key when no CREATE INDEX is suggested", () => {
    const text = buildCaptureTableText([buildRepository()])

    assert.equal(
      text.split("\n")[1],
      "1\t[micro-payment] Get user by PN\tselect id, username from user_data where pn = ?\tpn\t"
    )
    assert.doesNotMatch(text, /CREATE INDEX/)
  })

  test("has no group line for a single repository", () => {
    const text = buildCaptureTableText([buildRepository()])

    assert.doesNotMatch(text, /Layanan \/ Microservice/)
    assert.equal(text.split("\n").length, 2)
  })

  test("renders a single fallback row when no repository has a query", () => {
    const emptyRepositories = [buildRepository({ queries: [] })]

    const html = buildCaptureTableHtml(emptyRepositories)
    assert.match(
      html,
      /<tr><td colspan="5">Tidak ada penambahan atau perubahan query pada pengembangan kali ini\.<\/td><\/tr>/
    )
    assert.doesNotMatch(html, /Layanan \/ Microservice/)

    const text = buildCaptureTableText(emptyRepositories)
    assert.equal(
      text,
      "No\tFitur\tQuery Statement\tIndex\tCapture Explain / Execution Plan\n" +
        "Tidak ada penambahan atau perubahan query pada pengembangan kali ini."
    )
  })

  test("renders the fallback row when there are no repositories at all", () => {
    const html = buildCaptureTableHtml([])

    assert.match(html, /colspan="5"/)
    assert.match(html, /Tidak ada penambahan atau perubahan query pada pengembangan kali ini\./)
  })
})
