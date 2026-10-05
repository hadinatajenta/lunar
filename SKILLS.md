# SKILLS.md — Skills, Rules, and Agents

Index of the skills, rules, and agents available in the `lunar` repository, and the working
obligations that come with them.

---

## Standing obligation: keep the documentation current

This applies to every agent and every task, and it is enforced by `AGENTS.md` section 10.

**When you create, change, or remove a feature, a subfeature, or a flow, update `guide.md` in the
same change.** That means editing the owning section and adding a line to its change log. A feature
that exists in code but not in `guide.md` is undocumented. An entry in `guide.md` that no longer
exists in code is a defect.

Non-trivial features are planned under `plan/<feature>/` before implementation, with
`implementation_plan.md`, `task.md`, and `subagent_workstreams.md` when the work is parallelised.

---

## Repository skills

Defined in `.agents/skills/`, loaded on demand.

| Skill | File | Use when |
|---|---|---|
| `repo-engineering` | `.agents/skills/repo-engineering/SKILL.md` | Repository-wide tasks, discovery, instruction routing, full-stack workflows, contract synchronisation, whole-project verification, pre-commit diff review, and enforcing the safety guardrails |

`repo-engineering` establishes the instruction hierarchy:

```text
Root AGENTS.md  (global operating manual)
    |
    +-- opencode.jsonc  (tool permissions, safety rails, formatters)
    |
    +-- .agents/  (specialised agents, skills, and rules)
```

It also defines the eight-step lifecycle: understand, read applicable instructions, inspect existing
code, plan, implement, test, review the diff, report.

---

## Repository rules

Defined in `.agents/rules/`, always in force.

| Rule | File | Requires |
|---|---|---|
| No code comments | `.agents/rules/no_code_comments.md` | No inline, block, doc, or explanatory comments in any generated or edited code |
| Frontend English only | `.agents/rules/frontend_english_only.md` | All UI copy, templates, dialogs, and labels in English |
| Backend security and SAST | `.agents/rules/backend_security_sast.md` | SonarQube quality gate compliance and code-smell eradication on the backend |

These restate and extend sections 1, 2, 7, and 8 of `AGENTS.md`.

---

## Repository agents

Defined in `.agents/agents/`, each with its own scope and quality bar.

| Agent | File | Scope |
|---|---|---|
| `backend` | `.agents/agents/backend/agent.md` | Senior backend engineer: Go, Node.js, Express, Laravel, strict unit-test isolation with no real database connections, database safety, and anti-slop architecture |
| `frontend` | `.agents/agents/frontend/agent.md` | Senior frontend and UI/UX engineer: Vue, React, TypeScript, state management, routing, and anti-slop interface design |
| `code-reviewer` | `.agents/agents/code-reviewer/agent.md` | Read-only reviewer for git diffs, architecture boundaries, anti-slop patterns, security, and invariant compliance |

---

## Skills provided by the agent environment

These are supplied by the harness rather than stored in this repository. They may be loaded when the
task matches.

| Skill | Use when |
|---|---|
| `backend` | Backend implementation work |
| `frontend` | Frontend implementation work |
| `office-docx` | A Word document is an input or a deliverable |
| `office-pptx` | A PowerPoint deck is an input or a deliverable |
| `office-xlsx` | An Excel workbook is an input or a deliverable |

---

## Task routing

| Task | Route |
|---|---|
| Any coding task | `repo-engineering` skill, then the matching rule set |
| Backend change | `backend` agent, plus `backend_security_sast` and `no_code_comments` |
| Frontend change | `frontend` agent, plus `frontend_english_only` and `no_code_comments` |
| Feature planning | `plan/<feature>/` per `AGENTS.md` section 10.3 |
| After implementation | `code-reviewer` against the diff, and update `guide.md` |
| Document or spreadsheet deliverable | The matching `office-*` skill |

---

## Documentation map

| Document | Covers |
|---|---|
| `guide.md` | Every feature and subfeature, and the change log |
| `AGENTS.md` | Global operating manual and quality bar |
| `SKILLS.md` | This file |
| `docs/microservices/HELPER.md` | Helper installation, endpoints, and troubleshooting |
| `plan/BACKLOG.md` | Open items carried across features |
| `plan/<feature>/` | Per-feature plan, checklist, and workstream breakdown |
| `.agents/` | Rules, agents, and skills |
| `frontend/e2e/` | End-to-end specifications grouped by feature area |
