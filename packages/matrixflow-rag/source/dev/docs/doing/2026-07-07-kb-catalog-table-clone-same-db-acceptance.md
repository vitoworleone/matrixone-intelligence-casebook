# KB Catalog Table Same-DB Clone Review Fix Acceptance

## Goal

Verify PR #12789 review fix for KB catalog table append:

- succeeded structured-load table sources can still be reused by `kb_table_id`;
- failed or pending table sources are not treated as reusable just because `kb_table_id/db_name/table_name` are populated;
- failed/pending non-removed rows reuse the existing `source_id`, reset to pending, and continue through table-clone job creation;
- same-database table clone short-circuit remains intact.

Scope is limited to `moi-backend/pkg/session` plus the existing catalog clone behavior in `moi-backend/pkg/catalog`.

## Local Service Startup

Use the project local-deploy entry. Do not use `make start`, `make start-core`, `make start-backend`, or `make start-frontend` for AI-operated local startup.

From repo root:

```bash
bash skills/local-deploy/scripts/start-local.sh
```

The script starts local MatrixOne if needed, builds and starts Catalog, go-worker, custom-tool-worker, moi-backend, and moi-frontend under tmux.

Expected default endpoints:

| Service | Endpoint |
| --- | --- |
| Catalog HTTP | `http://127.0.0.1:8081` |
| moi-backend | `http://127.0.0.1:8050` |
| moi-frontend | `http://127.0.0.1:8000` |

If default ports are already occupied, the script automatically starts an isolated profile. Read the printed endpoints or `.runtime/<profile>/.env`.

Minimum health checks after startup:

```bash
curl --noproxy '*' http://127.0.0.1:8081/health
curl --noproxy '*' http://127.0.0.1:8050/health
```

Useful logs:

```text
.runtime/logs/catalog.log
.runtime/logs/go-worker.log
.runtime/logs/custom-tool-worker.log
.runtime/logs/moi-backend.console.log
.runtime/logs/moi-frontend.log
```

Local UI login:

```text
mobile: 13800000000
password: admin
```

## Code-Level Verification

Run from `moi-backend/`:

```bash
go test ./pkg/session -count=1 -run 'TestSemanticModelServiceAppendModelSources(ReusesStructuredLoadTableByKBTableID|DoesNotReuseFailedStructuredLoadTableByKBTableID)|TestSemanticModelServiceFindCatalogTableSourcePrefersSucceededReusableTable'
go test ./pkg/catalog -count=1 -run 'TestCloneTableForKnowledgeBaseSameDatabase|TestCloneTableForKnowledgeBaseFailsWhenTargetExists'
go test ./pkg/session ./pkg/catalog -count=1
```

Run from repo root:

```bash
git diff --check
```

## Manual Acceptance Scenario

After local services are up:

1. Open `http://127.0.0.1:8000` and log in as the local admin.
2. Create or open a knowledge base backed by a semantic model.
3. Add a Catalog table that already exists in the KB target database.
4. Confirm append completes without `already exists` clone failure.
5. Re-add a structured-load produced table whose source row is already `succeeded`.
6. Confirm it reuses the existing source and does not create a new table-clone job.
7. Seed or reproduce a failed non-removed `knowledge_base_sources` table row with populated `kb_table_id`, `db_name`, and `table_name`, then append the same table.
8. Confirm append does not return the failed row as success directly; the same `source_id` is reset to `pending`, a table-clone job is created or updated, and the row becomes `succeeded` only after clone/reconcile succeeds.

Do not verify by checking only UI success toast. Inspect source/job state through backend response, tenant DB, or logs.

## Current Verification Result

Completed on 2026-07-07 from current repo state:

```text
cd moi-backend && go test ./pkg/session -count=1 -run 'TestSemanticModelServiceAppendModelSources(ReusesStructuredLoadTableByKBTableID|DoesNotReuseFailedStructuredLoadTableByKBTableID)|TestSemanticModelServiceFindCatalogTableSourcePrefersSucceededReusableTable'
Result: passed, 3 tests.

cd moi-backend && go test ./pkg/catalog -count=1 -run 'TestCloneTableForKnowledgeBaseSameDatabase|TestCloneTableForKnowledgeBaseFailsWhenTargetExists'
Result: passed, 3 tests.

cd moi-backend && go test ./pkg/session ./pkg/catalog -count=1
Result: passed, 441 tests across 2 packages.

git diff --check
Result: passed.
```

Local service startup and manual UI/API acceptance were not run in this handoff. The command and checks above are the required local-service path for the next Codex verification pass.
