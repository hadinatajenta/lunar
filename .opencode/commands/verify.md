---
description: Validate the repository (tests, linting, build, git cleanliness) and report
---

Validate the `lunar` repository according to `AGENTS.md`. Do not modify code while verifying.

Execute the following checks and report the results:

1. Run automated test suites and linter.
2. Run build / compile commands if applicable.
3. Run `git status` to ensure no stray build artifacts, untracked temporary files, or uncommitted secrets exist.

$ARGUMENTS

Report a concise pass/fail summary per check and, for any failure, the exact error output.
