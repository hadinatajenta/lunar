import assert from "node:assert/strict"
import { describe, test } from "node:test"
import { isMarkdownSopBlock, sopBlockToText } from "./sop-block.ts"

describe("isMarkdownSopBlock", () => {
  test("treats storage markup as not markdown", () => {
    assert.equal(isMarkdownSopBlock("<p><strong>BAB I</strong></p>"), false)
    assert.equal(isMarkdownSopBlock('<ac:structured-macro ac:name="code"></ac:structured-macro>'), false)
  })

  test("treats plain prose as markdown and an empty block as neither", () => {
    assert.equal(isMarkdownSopBlock("## BAB I\nDeploy database"), true)
    assert.equal(isMarkdownSopBlock("   "), false)
  })
})

describe("sopBlockToText", () => {
  test("flattens Confluence storage into readable text without tags", () => {
    const storage =
      "<p><strong>BAB I. DEPLOY DATABASE MMS</strong></p><p>Tidak ada penambahan Table/Database</p><p><strong>BAB II</strong></p><ol><li>Build everest</li><li>Deploy to DC</li></ol>"

    const text = sopBlockToText(storage)

    assert.equal(text.includes("<"), false)
    assert.equal(text.includes(">"), false)
    assert.equal(text.includes("BAB I. DEPLOY DATABASE MMS"), true)
    assert.equal(text.includes("Build everest"), true)
    assert.equal(text.includes("Deploy to DC"), true)
  })

  test("decodes entities after stripping tags", () => {
    assert.equal(sopBlockToText("<p>a &amp; b &lt; c</p>"), "a & b < c")
  })
})
