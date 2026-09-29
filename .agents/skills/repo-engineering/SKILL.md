---
name: repo-engineering
description: >-
  Use this skill for repository-wide engineering tasks, discovery, instruction hierarchy routing,
  full-stack workflows, contract synchronization, whole-project verification, pre-commit diff review,
  and enforcing safety guardrails across lunar.
---

# Repository Engineering Skill

This skill provides operational routing, lifecycle management, and cross-cutting guardrails for the `lunar` repository.

---

## 1. Instruction & Documentation Hierarchy

Repository documentation is the authoritative source of truth. When planning or executing tasks, follow this precedence:

```text
Root AGENTS.md (Global Operating Manual: Architecture, Invariants, Quality Bar)
    │
    ├── opencode.jsonc (Tool Permissions, Safety Rails, Formatters)
    │
    └── .agents/ (Specialized Agents, Skills, and Rules)
```

---

## 2. Universal Engineering Lifecycle

For every coding task, execute this sequence:

1. **UNDERSTAND**: Clarify requirements, identify impacted layers, and determine scope boundaries.
2. **READ APPLICABLE INSTRUCTIONS**: Consult `AGENTS.md` and applicable rules in `.agents/rules/`.
3. **INSPECT EXISTING CODE**: Verify active implementations before making assumptions.
4. **PLAN**: Outline discrete changes, verify dependency direction, and prepare test strategies.
5. **IMPLEMENT**: Write minimal, cohesive, and idiomatic code adhering to conventions.
6. **TEST**: Execute deterministic automated CLI test suites and build steps.
7. **REVIEW DIFF**: Audit the final git diff against safety, zero comments, English-only, and anti-slop rules.
8. **REPORT**: Provide a concise summary of changes and verification results.

---

## 3. Repository Safety Guardrails

- **Zero Code Comments**: Never generate inline comments or block comments.
- **Do NOT run destructive Git commands**: Never execute `git push`, `git reset --hard`, `git clean -fd`, or commit changes without explicit user approval.
- **Do NOT edit or log secret credentials**: Never print access tokens, cookies, or secrets, and never overwrite `.env` files with dummy values.
- **Language**: English only for UI copy and code artifacts.
