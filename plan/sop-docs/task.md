# Task checklist

## Helper

- [x] `POST /sop/scan` with auto-stash and restore, same branch handling as `/ut-runs`
- [x] Changed database file detection (`.sql`, migrations, statement patterns), bounded
- [x] ORM migration warning when `AutoMigrate` is found
- [x] Tests: find statements, ignore unrelated files, bound output, stash restore

## Backend

- [x] `sop/domain`: repository scan, database change, SOP deploy/operasional, per-region block
- [x] Shared MIG reader that returns every repository, including `fe-mms` and `mms`
- [x] Managed block builder (BAB I-IV) with `code` macros and the `info` fallback
- [x] SOP Operasional plan (parent plus Maintenance/Monitoring/Troubleshooting skeletons)
- [x] `POST /api/confluence/documents/{id}/sop/generate`
- [x] `POST /api/confluence/documents/{id}/sop/apply` with per-chapter DB markers, per-region idempotent hash, version+1, 409 retry; BAB II/III written once and never replaced
- [x] Create pages under `<KEY>-Development`; create missing Operasional children only
- [x] Bamboo and database map config with placeholders and warnings
- [x] No screenshot generated and no attachment touched
- [x] Tests for the builder, marker replace, apply, conflict retry, children creation

## Frontend

- [x] Button "Generate SOP Deploy & Operasional" on the document detail view
- [x] SOP panel: preview of SOP Deploy and SOP Operasional, warnings, Copy, Apply
- [x] Helper `/sop/scan` orchestration per repository, then a second generate with the scans
- [x] Tests and Playwright

## Integration and documentation

- [x] `app.go` wiring and helper registration
- [x] `guide.md` section 5 and change log
- [ ] Live dry run against a real ticket before applying to a real page. Not run: the backend was not
      running and this needs a real Confluence page to write to
