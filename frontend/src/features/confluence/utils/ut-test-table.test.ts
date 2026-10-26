import assert from "node:assert/strict"
import { describe, test } from "node:test"
import type { UtDocsMockTest, UtDocsRow } from "../types.ts"
import {
  NO_UT_SCENARIO_MESSAGE,
  buildUtMockTestMarkdown,
  buildUtScenarioTableHtml,
  buildUtScenarioTableText,
  formatUtResultLabel,
  groupUtScenarioRows,
  hasExecutedUtRows,
  resolveUtResultTone
} from "./ut-test-table.ts"

const buildRow = (overrides: Partial<UtDocsRow> = {}): UtDocsRow => ({
  no: 1,
  repository: "micro-payment",
  function: "CreatePayment",
  scenario: "Creates a payment",
  expected_result: "The payment is stored",
  input_data: "valid payload",
  result: "Passed",
  screen_capture: "run-micro-payment.png",
  test_function_name: "TestCreate",
  ...overrides
})

const buildMockTest = (overrides: Partial<UtDocsMockTest> = {}): UtDocsMockTest => ({
  test_function_name: "TestCreatePaymentMock",
  function: "CreatePayment",
  scenario: "Creates a payment",
  expected_result: "The payment is stored",
  input_data: "valid payload",
  code: "func TestCreatePaymentMock(t *testing.T) {\n\tsqlmock.New()\n}",
  ...overrides
})

describe("resolveUtResultTone and formatUtResultLabel", () => {
  test("maps the three runner statuses case-insensitively", () => {
    assert.equal(resolveUtResultTone("Passed"), "passed")
    assert.equal(resolveUtResultTone("failed"), "failed")
    assert.equal(resolveUtResultTone("SKIPPED"), "skipped")
    assert.equal(resolveUtResultTone("pass"), "passed")
    assert.equal(formatUtResultLabel("passed"), "Passed")
    assert.equal(formatUtResultLabel("Failed"), "Failed")
    assert.equal(formatUtResultLabel("skipped"), "Skipped")
  })

  test("gives a Not Run row its own neutral tone instead of a success or failure", () => {
    assert.equal(resolveUtResultTone("Not Run"), "not_run")
    assert.equal(resolveUtResultTone("not run"), "not_run")
    assert.equal(resolveUtResultTone("not_run"), "not_run")
    assert.equal(formatUtResultLabel("Not Run"), "Not Run")
  })

  test("keeps an unknown result readable instead of inventing a status", () => {
    assert.equal(resolveUtResultTone("inconclusive"), "unknown")
    assert.equal(formatUtResultLabel("inconclusive"), "inconclusive")
    assert.equal(formatUtResultLabel(""), "Unknown")
  })
})

describe("hasExecutedUtRows", () => {
  test("is true when at least one row carries runner evidence", () => {
    assert.equal(
      hasExecutedUtRows([buildRow({ result: "Not Run" }), buildRow({ result: "Passed", no: 2 })]),
      true
    )
  })

  test("is false when every row is Not Run or unknown", () => {
    assert.equal(hasExecutedUtRows([buildRow({ result: "Not Run" })]), false)
    assert.equal(hasExecutedUtRows([buildRow({ result: "Not Run" }), buildRow({ result: "inconclusive", no: 2 })]), false)
    assert.equal(hasExecutedUtRows([]), false)
  })
})

describe("groupUtScenarioRows", () => {
  test("groups rows by repository in first-seen order", () => {
    const groups = groupUtScenarioRows([
      buildRow({ no: 1, repository: "micro-payment" }),
      buildRow({ no: 2, repository: "micro-notification" }),
      buildRow({ no: 3, repository: "micro-payment" })
    ])

    assert.equal(groups.length, 2)
    assert.equal(groups[0].repository, "micro-payment")
    assert.equal(groups[0].rows.length, 2)
    assert.equal(groups[1].repository, "micro-notification")
  })
})

describe("buildUtScenarioTableHtml", () => {
  test("renders the seven headers and one row per test function for a single repository", () => {
    const html = buildUtScenarioTableHtml([buildRow()])

    assert.match(html, /^<table class="wrapped relative-table" style="width: 100%;">/)
    assert.match(html, /<th>No<\/th>/)
    assert.match(html, /<th>Function<\/th>/)
    assert.match(html, /<th>Scenario<\/th>/)
    assert.match(html, /<th>Expected Result<\/th>/)
    assert.match(html, /<th>Input Data<\/th>/)
    assert.match(html, /<th>Result<\/th>/)
    assert.match(html, /<th>Screen Capture<\/th>/)
    assert.match(
      html,
      /<tr><td>1<\/td><td>CreatePayment<\/td><td>Creates a payment<\/td><td>The payment is stored<\/td><td>valid payload<\/td><td>Passed<\/td><td>run-micro-payment\.png<\/td><\/tr>/
    )
    assert.doesNotMatch(html, /Layanan \/ Microservice/)
  })

  test("groups more than one repository and renders the group row with the repository name", () => {
    const rows = [
      buildRow({ no: 1, repository: "micro-payment" }),
      buildRow({ no: 2, repository: "micro-payment", test_function_name: "TestUpdate" }),
      buildRow({ no: 1, repository: "micro-notification", test_function_name: "TestAudit" })
    ]

    const html = buildUtScenarioTableHtml(rows)

    assert.match(
      html,
      /<tr><td colspan="7"><strong>Layanan \/ Microservice: <code>micro-payment<\/code><\/strong><\/td><\/tr>/
    )
    assert.match(
      html,
      /<tr><td colspan="7"><strong>Layanan \/ Microservice: <code>micro-notification<\/code><\/strong><\/td><\/tr>/
    )
    const paymentGroup = html.split("micro-notification")[0]
    assert.match(paymentGroup, /<tr><td>1<\/td>/)
    assert.match(paymentGroup, /<tr><td>2<\/td>/)
    assert.match(html.split("micro-notification")[1], /<tr><td>1<\/td>/)
  })

  test("maps the runner result to a readable label and never shows a raw status", () => {
    const html = buildUtScenarioTableHtml([
      buildRow({ function: "A", result: "passed" }),
      buildRow({ function: "B", result: "FAILED", no: 2 }),
      buildRow({ function: "C", result: "skipped", no: 3 })
    ])

    assert.match(html, /<td>Passed<\/td>/)
    assert.match(html, /<td>Failed<\/td>/)
    assert.match(html, /<td>Skipped<\/td>/)
    assert.doesNotMatch(html, /<td>passed<\/td>/)
  })

  test("escapes every cell so a value cannot inject markup", () => {
    const html = buildUtScenarioTableHtml([
      buildRow({
        repository: 'pay"<script>',
        function: "<b>bold</b>",
        scenario: "a & b",
        expected_result: 'quote " and <tag>',
        input_data: "<script>alert(1)</script>",
        result: "passed",
        screen_capture: "<img src=x>"
      })
    ])

    assert.doesNotMatch(html, /<script>/)
    assert.doesNotMatch(html, /<b>bold<\/b>/)
    assert.doesNotMatch(html, /<img src=x>/)
    assert.match(html, /&lt;b&gt;bold&lt;\/b&gt;/)
    assert.match(html, /a &amp; b/)
    assert.match(html, /&lt;script&gt;alert\(1\)&lt;\/script&gt;/)
    assert.match(html, /&lt;img src=x&gt;/)
  })

  test("references the run screenshot per row in the Screen Capture cell", () => {
    const html = buildUtScenarioTableHtml([
      buildRow({ no: 1, screen_capture: "UT-101-run.png" }),
      buildRow({ no: 2, screen_capture: "UT-101-run.png", test_function_name: "TestUpdate" })
    ])

    const matches = html.match(/<td>UT-101-run\.png<\/td>/g) ?? []
    assert.equal(matches.length, 2)
  })

  test("renders a single fallback row when there is no test function", () => {
    const html = buildUtScenarioTableHtml([])

    assert.match(html, /<td colspan="7">No test function was reported for this run\.<\/td>/)
    assert.doesNotMatch(html, /Layanan \/ Microservice/)
  })
})

describe("buildUtScenarioTableText", () => {
  test("writes a header line and one line per function, grouped when there are several repositories", () => {
    const text = buildUtScenarioTableText([
      buildRow({ no: 1, repository: "micro-payment" }),
      buildRow({ no: 1, repository: "micro-notification", test_function_name: "TestAudit" })
    ])
    const lines = text.split("\n")

    assert.equal(lines[0], "No\tFunction\tScenario\tExpected Result\tInput Data\tResult\tScreen Capture")
    assert.equal(lines[1], "Layanan / Microservice: micro-payment")
    assert.equal(lines[2], "1\tCreatePayment\tCreates a payment\tThe payment is stored\tvalid payload\tPassed\trun-micro-payment.png")
    assert.equal(lines[3], "Layanan / Microservice: micro-notification")
    assert.equal(lines[4], "1\tCreatePayment\tCreates a payment\tThe payment is stored\tvalid payload\tPassed\trun-micro-payment.png")
  })

  test("has no group line for a single repository", () => {
    const text = buildUtScenarioTableText([buildRow()])

    assert.doesNotMatch(text, /Layanan \/ Microservice/)
    assert.equal(text.split("\n").length, 2)
  })

  test("writes the fallback message when there is no test function", () => {
    const text = buildUtScenarioTableText([])

    assert.equal(
      text,
      "No\tFunction\tScenario\tExpected Result\tInput Data\tResult\tScreen Capture\n" +
        NO_UT_SCENARIO_MESSAGE
    )
  })
})

describe("buildUtMockTestMarkdown", () => {
  test("fences each mock test as a go code block", () => {
    assert.equal(
      buildUtMockTestMarkdown([buildMockTest()]),
      "```go\nfunc TestCreatePaymentMock(t *testing.T) {\n\tsqlmock.New()\n}\n```"
    )
  })

  test("skips a mock test whose code is empty and returns nothing for an empty list", () => {
    assert.equal(buildUtMockTestMarkdown([]), "")
    assert.equal(buildUtMockTestMarkdown([buildMockTest({ code: "   " })]), "")
  })
})

describe("mock test code in the copied output", () => {
  test("appends each mock test as a pre code block after the table HTML", () => {
    const html = buildUtScenarioTableHtml([buildRow()], [buildMockTest()])

    assert.match(html, /<\/table><pre><code class="language-go">/)
    assert.match(html, /func TestCreatePaymentMock\(t \*testing\.T\)/)
  })

  test("escapes the mock code so it cannot inject markup", () => {
    const html = buildUtScenarioTableHtml(
      [buildRow()],
      [buildMockTest({ code: "if a < b && c > d {}" })]
    )

    assert.match(html, /if a &lt; b &amp;&amp; c &gt; d \{\}/)
    assert.doesNotMatch(html, /<b /)
  })

  test("appends each mock test as a fenced block after the table text", () => {
    const text = buildUtScenarioTableText([buildRow()], [buildMockTest()])

    assert.match(text, /\n\n```go\nfunc TestCreatePaymentMock/)
  })

  test("keeps the table text unchanged when there is no mock test", () => {
    assert.equal(buildUtScenarioTableText([buildRow()]).split("\n").length, 2)
  })

  test("writes the mock code after the empty fallback text", () => {
    assert.equal(
      buildUtScenarioTableText([], [buildMockTest()]),
      "No\tFunction\tScenario\tExpected Result\tInput Data\tResult\tScreen Capture\n" +
        NO_UT_SCENARIO_MESSAGE +
        "\n```go\nfunc TestCreatePaymentMock(t *testing.T) {\n\tsqlmock.New()\n}\n```"
    )
  })
})
