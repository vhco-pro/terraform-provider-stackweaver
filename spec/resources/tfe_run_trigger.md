<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_run_trigger
tfe_alias: tfe_run_trigger
kind: resource
family: runs
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_run_trigger.go
go_tfe_type: RunTrigger
compat_doc: docs/internal/tfe-compatibility/resources/runs/tfe_run_trigger.md
---
# stackweaver_run_trigger

Links a **source** workspace to a **target** workspace so that a successful apply in the source
auto-queues a plan-and-apply run in the target. Maps 1:1 onto Stackweaver's run-trigger concept
(`core/models/run_trigger.go`, one row per source→target pair). The trigger fires at run time via the
orchestrator, not just at plan/apply of this resource.

## Client approach

`go-tfe-clean`. Stackweaver's run-triggers endpoints accept and return the stock `go-tfe` `RunTrigger`
JSON:API shape unchanged (`docs/internal/tfe-compatibility/resources/runs/tfe_run_trigger.md`); no
wrapper. The upstream resource (legacy SDKv2, `internal/provider/resource_tfe_run_trigger.go:22`) drives
the `go-tfe` `RunTriggers` service verbatim (`Create`, `ReadWithOptions`, `Delete`), sending the
`sourceable` relation and reading back the `workspace` (target) + `sourceable` (source) relations.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | `run-triggers` JSON:API primary id (`rt-…`) |
| `workspace_id` | string | Required | yes | — | no | the **target**: workspace whose runs get queued; also the create-path workspace |
| `sourceable_id` | string | Required | yes | — | no | the **source**: workspace whose applies fire the trigger; must be in the same org |

## Wire contract

- **Create:** `RunTriggers.Create(workspace_id, RunTriggerCreateOptions{Sourceable: &Workspace{ID: sourceable_id}})`
  → `POST /workspaces/:workspace_id/run-triggers`. Body carries the `sourceable` relation
  (`{"data":{"type":"workspaces","id":"<sourceable_id>"}}`); path workspace is the target. Wrapped in a
  1-minute retry on the "Run Trigger creation locked" transient error (upstream behavior).
- **Read:** `RunTriggers.ReadWithOptions(id, {Include: [workspace, sourceable]})` → `GET /run-triggers/:id`;
  sets `workspace_id` from the `workspace` relation and `sourceable_id` from the `sourceable` relation.
- **Update:** none — both attributes are ForceNew, so any change recreates.
- **Delete:** `RunTriggers.Delete(id)` → `DELETE /run-triggers/:id` (idempotent; a 404 is treated as
  already gone).
- **JSON:API type:** `run-triggers`. Computed convenience attrs `workspace-name` / `sourceable-name` and
  `created-at` are read from the wire but not surfaced as schema attributes. No write-only fields.

## Acceptance criteria (these ARE the test)

Concrete, testable — the `implement` pipeline generates fixture assertions from these.

1. `apply` of a fixture with two workspaces + `{workspace_id (target), sourceable_id (source)}` creates
   the trigger; `id` (`rt-…`), `workspace_id`, and `sourceable_id` round-trip into state.
2. Re-`plan` after apply shows **no drift**.
3. On read, the source appears in the target's **inbound** run-triggers list
   (`GET /workspaces/:target/run-triggers?filter[run-trigger][type]=inbound`).
4. Changing `workspace_id` or `sourceable_id` recreates (both ForceNew) — plan shows destroy+create,
   not in-place update.
5. `destroy` removes it; a subsequent `RunTriggers.Read(id)` returns 404 and the source is absent from
   the target's inbound list.
6. Same-org / no-self-trigger: source and target must be in the same org, and a workspace triggering
   itself is rejected (matches TFE tenant safety).

## Runtime criterion

The trigger fires — this is the feature, not CRUD. With a source→target trigger in place, applying a
run in the **source** causes the orchestrator (`backend/cmd/orchestrator/main.go` `processRunTriggers`,
~10s tick) to queue a **plan-and-apply** run in the **target** within one worker tick, using the
target's latest configuration version and respecting the target's `auto_apply`. Each applied source run
fires its downstream runs **exactly once** (the `run_triggers_fired_at` atomic claim). A source run with
no trigger queues nothing. Verified live end-to-end, independent of whether a remote or agent runner
applied the source.

## Docs + example

- Provider docs page: `docs/resources/run_trigger.md` — arguments `workspace_id` (target) and
  `sourceable_id` (source), computed `id`, the same-org constraint, import by id, and the cycle caveat.
- Example: `examples/resources/stackweaver_run_trigger/resource.tf` — two workspaces and a
  `stackweaver_run_trigger` wiring source→target.

## Divergences from upstream / TFE

None. Drop-in with `tfe_run_trigger`. **Cycle caveat (matches TFE):** A↔B mutual triggers can loop —
each apply fires once, but the triggered run in B will itself trigger A. TFE has the identical footgun;
avoid trigger cycles. This is shared behavior, not a Stackweaver divergence.
