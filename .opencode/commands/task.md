---
description: Run the repository standard 6-stage engineering workflow for a task
---

You are working directly inside the `lunar` repository. Follow the repository established workflow and treat `AGENTS.md` as authoritative.

Requested task:
$ARGUMENTS

## 1. Classify the task and read the relevant instructions
- Read `AGENTS.md` to understand conventions, patterns, and guardrails.
- Use read tools directly; never ask the user to paste contents that exist in the workspace.

## 2. Inspect before changing
- Read the existing implementation, its callers, dependencies, and test suites.
- Verify assumptions against source code, not memory.

## 3. Plan
- State the plan and the files you expect to create or modify.
- Identify existing code, utilities, or types to reuse.
- Highlight edge cases and potential failure modes.

## 4. Implement
- Make the smallest change consistent with existing patterns and naming standards.
- Enforce the **Zero Code Comments** mandate.
- Keep all UI text and code identifiers strictly in **English**.
- Do not refactor unrelated code.

## 5. Test and validate
- Run relevant unit tests, build commands, and type checks.
- Verify test coverage reflects real behaviors, not fake assertions.

## 6. Review the diff and report
- Run `git diff`, re-read changed lines, and confirm all invariants are preserved.
- Report: files changed, why each changed, commands executed and their output.
- Never commit, push, or discard user changes unless explicitly asked.
