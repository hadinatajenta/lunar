# SOP docs: generate and apply SOP Deploy and SOP Operasional

## Objective

From a wiki page opened from a Jira ticket, one action generates and writes the two SOP documents the
project already uses: `<KEY>-SOP Deploy` and `<KEY>-SOP Operasional`, both as children of
`<KEY>-Development`. Content that we do not generate is never overwritten.

## Findings (from the real MCP space, Fortune 2026)

Verified against all 26 tickets in Sprints #34 to #44 and the filled examples (CRMMS-71000, CRMMS-67778,
CRMMS-66351).

- `<KEY>-Development` children: `[YYYYMMDD] Dokumen Query Review`, `Code Review Documentation`,
  `Hasil Meeting Design Review`, `IT Risk Control`, `MIG`, `SIT/UIT`, `SOP Deploy`, `SOP Operasional`,
  `User Manual`, `UT`. `SOP Deploy` and `SOP Operasional` are siblings of `MIG` and `UT`.
- `SOP Deploy` is a leaf with a fixed skeleton written as bold paragraphs, not headings:
  `BAB I. DEPLOY DATABASE MMS`, `BAB II. DEPLOY APLIKASI` (sometimes `BAB II DEPLOYMENT SERVICE`),
  `BAB III. ROLLBACK APLIKASI`, and in larger tickets `BAB IV ROLLBACK DATABASE`.
  - BAB I: an `info` macro `Tidak ada penambahan Table/Database` when the change adds no table, or a
    `No | IP | Database` table, an ordered list of steps, and one `code` macro per SQL statement.
  - BAB II: ordered steps to build each repository in Bamboo for `release/production`, deploy to the
    DC environment, check pods on OCP, sometimes a CronJob YAML in a `code` macro, and screenshots as
    `ac:image` attachments.
  - BAB III: Bamboo plan selection, `promote existing`, delete the CronJob.
  - BAB IV: a `code` macro per `DROP TABLE IF EXISTS` for the tables created in BAB I.
- `SOP Operasional` is a container with only a `children` macro and exactly three children:
  `Maintenance` (TOC macro plus `1. Backup`, `2. Restore`, `3. Housekeeping`), `Monitoring` (OCP steps),
  and `Troubleshooting` (usually empty).
- Newest tickets are still 147-character stubs, so the feature has to fill the skeleton, not invent a
  new shape.

## Entry point

The document detail view (the wiki page opened from a Jira ticket) gains a button
**Generate SOP Deploy & Operasional**. The backend resolves the issue key from the document, or from the
document ancestors, and uses the `<KEY>-Development` page as the parent.

## Source of truth: deterministic, never invented

- SQL and DDL come from the changed `.sql` and migration files on the branch, read by the helper. The
  model never writes a SQL statement.
- The repository list comes from the MIG page, and for SOP it includes every repository, including
  `fe-mms` and `mms` (the query review exclusion does not apply here).
- The Bamboo plan per repository and the database host and name are not derivable with certainty today.
  They come from an optional configuration map (`LUNAR_SOP_BAMBOO_MAP`, `LUNAR_SOP_DATABASE_MAP`, JSON),
  and when a value is missing it is written as an explicit placeholder and reported in `warnings`.
- The model is optional and, when used, only groups changes per service and writes prose. It is never the
  source of a statement, a table name, or a result.

## Contracts

### Helper `POST /sop/scan` (Bearer token, sync)

Request `{ "root", "repo", "branch", "base_ref" }`. The helper checks the branch out with the same
auto-stash and restore behavior as the UT run, then scans the changed files for database changes.

Response:

```json
{
  "repository": "everest",
  "branch": "feature/CRMMS-77914",
  "base_ref": "master",
  "database_files": [
    { "path": "migrations/2026_add_mdr.sql", "statements": [ { "action": "CREATE", "table": "log_program_mdr", "sql": "CREATE TABLE ..." } ] }
  ],
  "tables": [ { "name": "log_program_mdr", "action": "CREATE" } ],
  "warnings": ["GORM AutoMigrate found in internal/repo/store.go; DDL is not read from ORM code"]
}
```

SQL detection is bounded: `.sql` files, `migrations`/`migration`/`database/migrations` directories, and
`CREATE|ALTER|DROP|INSERT|UPDATE|DELETE` statements. Each statement is capped; the total is capped.

### `POST /api/confluence/documents/{id}/sop/generate`

Request `{ "model": "..." }` (model optional). Response:

```json
{
  "issue_key": "CRMMS-78064",
  "development_page_id": "2144721483",
  "repositories": [ { "repository": "everest", "branch": "feature/CRMMS-77914", "database_files": [], "warnings": [] } ],
  "deploy": { "title": "CRMMS-78064-SOP Deploy", "page_id": "2144721548", "exists": true, "managed_block": "<storage html>" },
  "operasional": { "title": "CRMMS-78064-SOP Operasional", "page_id": "2144721557", "exists": true, "children": [ { "title": "Maintenance", "exists": true }, { "title": "Monitoring", "exists": false }, { "title": "Troubleshooting", "exists": false } ] },
  "warnings": ["no Bamboo plan configured for everest"]
}
```

The managed block is the SOP Deploy body: BAB I with `info` or the DB table plus one `code` macro per
statement, BAB II and BAB III from the repository and Bamboo map, and BAB IV `DROP TABLE` for every table
created in BAB I.

### `POST /api/confluence/documents/{id}/sop/apply`

Request `{ "deploy": { "mode": "create|update" }, "operasional": { "create_children": true } }`.

Response `{ "deploy_page": { "id", "version", "url", "action" }, "operasional_page": { "id", "version", "url" }, "created_children": ["Monitoring","Troubleshooting"] }`.

## Handling existing pages

- **Managed region per database chapter.** Only the two database chapters are auto-managed:
  `<p><em>[[lunar:sop:db:start]]</em></p>` around `BAB I`, and
  `<p><em>[[lunar:sop:rollback-db:start]]</em></p>` around `BAB IV`.
- `BAB II. DEPLOY APLIKASI` and `BAB III. ROLLBACK APLIKASI` are written once, when the page is created
  or prepended, and are **never replaced by a later apply**. They are where the user adds the manual
  Bamboo and OCP screenshots, so a re-apply must not touch them.
- Page missing: create it as a child of `<KEY>-Development` with BAB I, II, III, IV and the two markers.
- Page present without markers: prepend the whole block and keep every existing node below it, then add
  the markers.
- Page present with markers: replace only the `BAB I` and `BAB IV` regions; leave BAB II and III and
  everything else untouched.
- **No screenshot is generated.** Bamboo and OCP screenshots are environment-specific and cannot be
  captured, so the generated block contains steps only, plus a note that the screenshots are added
  manually in BAB II and III.
- **Idempotency.** A per-region content hash is stored; an unchanged region is skipped and the version is
  not bumped.
- **Version conflict.** Update with `version.number + 1`. On `409`, refetch and retry once; a second
  failure is a `502` with a safe message.
- **Attachments.** No attachment is uploaded or removed by this feature. Existing attachments are never
  touched.
- **SOP Operasional children.** Create the three children only when they are missing, never overwrite an
  existing child.

## Module layout

```text
backend/internal/sop/
  domain/          SOP documents, database change, repository scan, managed block
  application/     generate, managed-block build, marker replace, apply orchestration
  infrastructure/  model client, bamboo and database maps, page writer, hash store
  transport/       HTTP handlers

backend/cmd/lunar-helper/handlers_sop.go   POST /sop/scan
```

Frontend, inside the Confluence feature: the button on the document detail view, a SOP panel with the
preview, warnings, Copy, and Apply, plus the helper scan call.

## Stages

1. Helper `/sop/scan`: changed database files and statements, bounded, no DB.
2. Backend `sop`: generate (deterministic block), shared MIG reader that includes every repository.
3. Backend apply: create/update with markers, idempotent hash, version+1 and 409 retry, SOP Operasional
   children.
4. Frontend: button, panel, Copy, Apply.
5. Documentation: `guide.md` section 5 and change log.

## Verification

- `go build ./...`, `go test ./...`, `go vet`, `gofmt` on the touched packages.
- `vue-tsc`, `vite build`, unit and Playwright confluence.
- A live dry run: generate against a real ticket, review the preview, and apply to a scratch page before
  a real one.

## Risks

| Risk | Mitigation |
|---|---|
| Overwriting manual SOP work | Managed-region markers and prepend, never a whole-body replace |
| A wrong SQL statement reaching Confluence | Statements come from the repo files verbatim; the model never writes SQL |
| Missing Bamboo or database mapping | Explicit placeholder plus `warnings` in the panel |
| Page written under the wrong parent | The parent is always resolved as `<KEY>-Development` from the page tree |
| Confluence version conflict | Refetch and retry once, then a safe `502` |
