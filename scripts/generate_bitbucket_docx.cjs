const fs = require("fs")
const path = require("path")
const { Document, Packer, Paragraph, TextRun, ImageRun, HeadingLevel, AlignmentType } = require("/Users/erendt/code/lunar/frontend/node_modules/docx")

function createHeading(text, level = HeadingLevel.HEADING_1) {
  return new Paragraph({
    text: text,
    heading: level,
    spacing: { before: 240, after: 120 }
  })
}

function createParagraph(text) {
  return new Paragraph({
    children: [new TextRun({ text, font: "Calibri", size: 22 })],
    spacing: { after: 120 }
  })
}

function createImage(imagePath, caption) {
  const fullPath = path.resolve(imagePath)
  if (!fs.existsSync(fullPath)) return []

  const imageBuffer = fs.readFileSync(fullPath)
  return [
    new Paragraph({
      alignment: AlignmentType.CENTER,
      children: [
        new ImageRun({
          data: imageBuffer,
          transformation: {
            width: 580,
            height: 340
          }
        })
      ],
      spacing: { before: 180, after: 80 }
    }),
    new Paragraph({
      alignment: AlignmentType.CENTER,
      children: [
        new TextRun({
          text: caption,
          italics: true,
          font: "Calibri",
          size: 18,
          color: "666666"
        })
      ],
      spacing: { after: 200 }
    })
  ]
}

async function generateBitbucketDocx() {
  const doc = new Document({
    sections: [
      {
        properties: {},
        children: [
          createHeading("Lunar QA Automation: Bitbucket Code Review Workspace", HeadingLevel.TITLE),
          createParagraph("Target: http://localhost:5173/bitbucket | Status: PASSED (100%) | Date: 2026-09-29"),
          createHeading("Test Scenarios & Results", HeadingLevel.HEADING_2),
          createParagraph("1. Bitbucket Code Review Workspace Overview Rendering - Passed"),
          createParagraph("2. Pushed Branches Filtering (Ready, Stale, All) - Passed"),
          createParagraph("3. Open Pull Requests Filtering (Assigned to me, AI Flagged, All) - Passed"),
          createParagraph("4. Create Pull Request Modal Submission & Toast Notification - Passed"),
          createParagraph("5. PR Review Modal, Git Diff Inspection, AI Review Synthesis & Comment Prefill - Passed"),
          createParagraph("6. PR Review Status Actions (Approve, Needs Work, Decline) - Passed"),
          createParagraph("7. Custom Comment Submission & Confirmation Toast - Passed"),
          createParagraph("8. Clean Empty States (Zero Pushes, Zero PRs) - Passed"),
          createParagraph("9. [Negative] Invalid or Expired Bitbucket PAT (HTTP 401) Error Banner & Settings Link - Passed"),
          createParagraph("10. [Negative] VPN Disconnect / Gateway Timeout (HTTP 502) Guidance Banner - Passed"),
          createParagraph("11. [Negative] Repository Access Forbidden (HTTP 403) Permission Warning - Passed"),
          createParagraph("12. [Negative] PAT Unconfigured Warning Banner with Settings Redirection - Passed"),
          createHeading("Visual Evidence", HeadingLevel.HEADING_2),
          ...createImage("/Users/erendt/code/lunar/docs/bitbucket/evidence/01_bitbucket_overview.png", "Figure 1: Bitbucket Workspace Overview"),
          ...createImage("/Users/erendt/code/lunar/docs/bitbucket/evidence/02_pushes_filter_ready.png", "Figure 2: Filter Pushed Branches: Ready Only"),
          ...createImage("/Users/erendt/code/lunar/docs/bitbucket/evidence/03_prs_filter_ai_flagged.png", "Figure 3: Filter Pull Requests: AI Flagged"),
          ...createImage("/Users/erendt/code/lunar/docs/bitbucket/evidence/04_create_pr_modal.png", "Figure 4: Create Pull Request Modal"),
          ...createImage("/Users/erendt/code/lunar/docs/bitbucket/evidence/05_create_pr_toast.png", "Figure 5: Pull Request Created Toast Notification"),
          ...createImage("/Users/erendt/code/lunar/docs/bitbucket/evidence/06_pr_review_modal_empty.png", "Figure 6: PR Review Modal (Initial State)"),
          ...createImage("/Users/erendt/code/lunar/docs/bitbucket/evidence/07_pr_review_ai_generated.png", "Figure 7: AI Code Review Synthesis & Comment Population"),
          ...createImage("/Users/erendt/code/lunar/docs/bitbucket/evidence/08_pr_review_status_action.png", "Figure 8: Review Approval Status Action"),
          ...createImage("/Users/erendt/code/lunar/docs/bitbucket/evidence/09_pr_review_comment_posted.png", "Figure 9: Custom Review Comment Posted"),
          ...createImage("/Users/erendt/code/lunar/docs/bitbucket/evidence/10_bitbucket_empty_state.png", "Figure 10: Clean Empty State for Pushes and Pull Requests"),
          ...createImage("/Users/erendt/code/lunar/docs/bitbucket/evidence/11_bitbucket_invalid_pat_negative.png", "Figure 11: Negative Test - Invalid/Expired Bitbucket PAT Banner"),
          ...createImage("/Users/erendt/code/lunar/docs/bitbucket/evidence/12_bitbucket_vpn_disconnect_negative.png", "Figure 12: Negative Test - VPN Disconnect Guidance Banner"),
          ...createImage("/Users/erendt/code/lunar/docs/bitbucket/evidence/13_bitbucket_access_forbidden_negative.png", "Figure 13: Negative Test - Access Forbidden Permission Warning"),
          ...createImage("/Users/erendt/code/lunar/docs/bitbucket/evidence/14_bitbucket_no_pat_configured.png", "Figure 14: Negative Test - PAT Unconfigured Warning Banner")
        ]
      }
    ]
  })

  const buffer = await Packer.toBuffer(doc)
  fs.writeFileSync("/Users/erendt/code/lunar/docs/bitbucket/QA_REPORT.docx", buffer)
  console.log("Generated /Users/erendt/code/lunar/docs/bitbucket/QA_REPORT.docx")
}

generateBitbucketDocx().catch(console.error)
