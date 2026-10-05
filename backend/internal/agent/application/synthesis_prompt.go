package application

const fenceMarker = "```"

const defaultSynthesisPrompt = `Based on all the code and dependency analysis above, produce a comprehensive, structured, and accurate answer. Include service names, endpoints, broker topics, and the request flow step by step.

DIAGRAMS ARE REQUIRED. When the answer describes a flow, an ordering of calls, or an architecture, you MUST render it as a mermaid diagram inside a fenced block whose language is exactly mermaid. Write the fence as three backticks followed by the single word mermaid.

This exact shape is required:

` + fenceMarker + `mermaid
---
title: Sequence of QRIS bulk detail approval
---
sequenceDiagram
    participant FE as fe-mms
    participant API as mms-api
    participant SVC as serviceQris
    FE->>API: POST /v1/cm-qris/approval-bulk
    API->>API: verifyToken, IsValidAsmenDepthead
    API->>SVC: RegisterUpdateToDbQris
    SVC-->>API: ok
    API-->>FE: 200
` + fenceMarker + `

Never do this. A plain arrow list is not a diagram and will be shown as an unreadable code block:

` + fenceMarker + `flow
POST /v1/cm-qris/approval-bulk -> ApprovalCmBulkQrisStatic -> Service.ApprovalCmBulkQrisStatic
` + fenceMarker + `

Rules:
- Use sequenceDiagram for the ordered interaction between services. Put the HTTP method and path on the arrow.
- Use "flowchart TD" for a control flow, a decision path, or a dependency chain. TD means top to bottom, which is the expected orientation.
- Use classDiagram for types and structs, and erDiagram for tables and their relations.
- Never invent a fence language. The only accepted language for a diagram is mermaid. Not flow, not text, not plain.
- Never describe a flow as inline arrow text such as a -> b -> c outside a mermaid block.
- Quote any node label that contains punctuation, parentheses, or a slash.
- You may include more than one diagram when the answer covers more than one flow. Each goes in its own mermaid block.

- Begin every mermaid block with a title in frontmatter, as shown above, so each diagram carries a readable name. Keep the title under eight words.

CODE BLOCKS: always put the language right after the fence, and when only a few lines matter, add the line numbers in braces after it. The renderer highlights exactly those lines.

` + fenceMarker + `go {3,5-7}
func handle() error {
    value, err := load()
    if err != nil {
        return err
    }
    return nil
}
` + fenceMarker + `

Use this to point at the line that carries the behaviour being explained, rather than pasting a long file.
Do not use LaTeX mathematical notation; use -> or the arrow character instead.`
