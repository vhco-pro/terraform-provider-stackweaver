<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_workspace_run
tfe_alias: tfe_workspace_run
kind: resource
family: runs
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_workspace_run.go
go_tfe_type: RunCreateOptions / RunApplyOptions
compat_doc: docs/internal/tfe-compatibility/resources/runs/tfe_workspace_run.md
---
# stackweaver_workspace_run

A **lifecycle-only** resource with no stored object of its own: on `create` it starts an apply run in
`workspace_id` and (by default) waits for it to reach `applied`; on `destroy` it starts a destroy run
and waits; `update` is a no-op. It exists to sequence runs between workspaces (e.g. apply a networking
workspace, wait, then apply an app workspace) without a VCS trigger. It orchestrates Stackweaver's
existing runs API - there is no by-id `workspace-runs` endpoint.

## Client approach

`go-tfe-clean`. The upstream resource (legacy SDKv2,
`internal/provider/resource_tfe_workspace_run.go:73` + `workspace_run_helpers.go`) is pure client-side
orchestration over the `go-tfe` `Runs` service (`Runs.Create`, `Runs.Read`, `Runs.Apply`); the nested
`apply {}` / `destroy {}` blocks are all **provider-side** poll/confirm/retry tuning that never reach
the wire. Stackweaver accepts stock go-tfe `RunCreateOptions` / `RunApplyOptions` unchanged - no
wrapper. Making it work end-to-end required closing three server-side run-create gaps (see wire
contract); each fix helps *every* go-tfe caller, not just this resource, and none change the wire the
provider sends.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | - | - | no | synthetic: the apply run id, or a random int for a destroy-only resource |
| `workspace_id` | string | Required | yes | - | no | workspace the run executes in |
| `apply` | list(block) max 1 | Optional | no | - | no | at-least-one-of `apply`/`destroy`; drives a create-time apply run |
| `destroy` | list(block) max 1 | Optional | no | - | no | drives a delete-time destroy run (`is-destroy: true`) |
| `apply/destroy.manual_confirm` | bool | Required (in block) | no | - | no | `false` → provider auto-confirms via `POST /runs/:id/actions/apply`; `true` → waits for out-of-band confirm |
| `apply/destroy.message` | string | Optional | no | - | no | run `message` on create |
| `apply/destroy.retry` | bool | Optional | no | `true` | no | provider-side: retry errored runs |
| `apply/destroy.retry_attempts` | int | Optional | no | `3` | no | provider-side |
| `apply/destroy.retry_backoff_min` | int | Optional | no | `1` | no | provider-side (seconds) |
| `apply/destroy.retry_backoff_max` | int | Optional | no | `30` | no | provider-side (seconds) |
| `apply/destroy.wait_for_run` | bool | Optional | no | `true` | no | provider-side: poll to completion; `false` = fire-and-forget (relies on server-side `auto-apply`) |

## Wire contract

- **Create (apply block):** `Runs.Create(RunCreateOptions{Workspace, Message?})` → `POST /runs`, then
  poll `Runs.Read(id)` → `GET /runs/:id` until `planned`, confirm via `Runs.Apply(id)` →
  `POST /runs/:id/actions/apply`, then poll to `applied`/`errored`. Standard go-tfe **`auto-apply`** and
  **`plan-only`** wire attrs are honored server-side (a plain go-tfe run resolves to an applyable
  `plan-and-apply`).
- **Create (destroy block, on resource delete):** same, with `IsDestroy: true` → `is-destroy` on the
  wire.
- **Read:** for an apply resource, `Runs.Read(id)` → `GET /runs/:id`; a destroy-only resource has no
  server object to read (returns without touching state). No `workspace-runs` endpoint exists.
- **Update:** no-op (provider `Update` returns nil).
- **Delete:** starts a destroy run via the same `Runs.Create`/poll/confirm flow.
- **JSON:API type:** `runs`. Server-side compatibility fixes this resource depends on:
  `resolveRunOperation` binds go-tfe's real `auto-apply`+`plan-only` (the non-standard `operation` /
  `auto-apply-after-plan` bindings remain as **additive fallbacks** for the frontend, not what the
  provider sends); a run with no configuration-version falls back to the workspace's **latest**;
  `GET /runs/:id/task-stages` returns an empty paginated list (no run-tasks subsystem); `is-confirmable`
  mirrors `permissions.can-apply`; `has-changes` counts destroy operations. No write-only fields.

## Acceptance criteria (these ARE the test)

Concrete, testable - the `implement` pipeline generates fixture assertions from these.

1. `apply` of a fixture (workspace with an uploaded config, `apply { manual_confirm = false }`) starts a
   real run that reaches **`applied`**; `id` is set to the apply run id and round-trips into state.
2. `plan -detailed-exitcode` after apply shows **no drift** (exit 0).
3. `workspace_id` is ForceNew - changing it recreates.
4. `destroy` with a `destroy { manual_confirm = false }` block starts a **destroy run** (`is-destroy`)
   that reaches `applied`; after that the apply run id is gone from state.
5. `update` is a no-op - changing a provider-side field (e.g. `message`) does not start a new run.
6. `wait_for_run = true` (default) apply-and-wait: the provider polls `is-confirmable`, confirms via
   `POST /runs/:id/actions/apply`, and the run reaches `applied` (does not hang at `planned`).
7. `wait_for_run = false, manual_confirm = false` fire-and-forget: the run auto-applies server-side via
   `auto-apply:true` and does not hang at `planned`.

## Runtime criterion

The run **is** the behavior - not CRUD. Verified live: creating the resource starts a real run that
plans, waits at `planned`, is confirmed, and reaches `applied`; destroying it starts a destroy run that
reaches `applied`. The apply-and-wait confirmation and the destroy path both exercise the real
`POST /runs/:id/actions/apply` transition against a runner (remote or agent).

## Docs + example

- Provider docs page: `docs/resources/workspace_run.md` - `workspace_id`, the `apply {}` / `destroy {}`
  blocks and their `manual_confirm` / `message` / `wait_for_run` / retry fields, the lifecycle-only /
  no-import nature, and the note that `refresh` / `refresh_only` / `target_addrs` / `allow_empty_apply`
  run options are not modelled by Stackweaver runs.
- Example: `examples/resources/stackweaver_workspace_run/resource.tf` - a workspace + a
  `stackweaver_workspace_run` with `apply { manual_confirm = false }` and
  `destroy { manual_confirm = false }`.

## Divergences from upstream / TFE

None (drop-in for the common `apply {}` / `destroy {}` usage with `manual_confirm` and default
`wait_for_run = true`). The `operation` / `auto-apply-after-plan` wire bindings are Stackweaver-side
**additive fallbacks** for the frontend and are *not* used by this resource - the provider sends stock
go-tfe `auto-apply` / `plan-only` / `is-destroy`. `refresh`, `refresh_only`, `target_addrs`, and
`allow_empty_apply` inside the triggered run are not yet modelled by Stackweaver runs; this resource
does not set them, so they are documented as unsupported rather than silently dropped.
