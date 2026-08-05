<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
# Tasks — terraform-provider-stackweaver v0.1

The ordered, dependency-aware worklist the automated `implement` pipeline consumes. Derived from
`plan/plan.md` (the HOW) over `spec/` (the WHAT). Each resource task = register `stackweaver_*` +
`tfe_*` → build → generate fixture from the spec's acceptance criteria → `dev_overrides` acceptance
(loop-until-green) → runtime-verify → docs+example → one PR.

## T0 — Provider plumbing (once, first; blocks everything)

Single foundational task (`plan.md` §1–4), one PR:
1. Provider address → `registry.terraform.io/vhco-pro/stackweaver`; primary type name `stackweaver`.
2. `alias.go`: generic `aliasResource`/`aliasDataSource` wrappers (framework) + dual-key registration
   (SDKv2). Register the 39 resources + 22 data sources under both `stackweaver_*` and `tfe_*`.
3. Strip the `dropped` set + the partial/blocked rows from the registration lists (keep files).
4. `internal/stackweaver/` skeleton (Phase-5 native client; one reference method).
5. Client host defaults / UA → Stackweaver. `go build` + `go test ./...` green.

Gate: provider builds, `terraform providers` lists both prefixes, no unsupported types exposed.

## Dependency graph (why the order below)

Fixtures need their referenced objects to exist first. Ordering rules:
- **org** exists in the dev stack (`dev-test`) already — no task; everything hangs off it.
- `project` → before `project_settings`, `project_variable_set`, `project_notification_configuration`.
- `workspace` → before `workspace_settings`, `workspace_run`, `workspace_run_task`, `run_trigger`
  (source+target), `workspace_variable_set`.
- `agent_pool` → before `agent_pool_allowed_*`, `agent_pool_excluded_workspaces`, `agent_token`.
- `team` → before `team_access`, `team_members`, `team_organization_member(s)`, `team_project_access`,
  `team_token`, `team_notification_configuration`.
- `variable_set` → before `workspace_variable_set`, `project_variable_set`.
- `organization_run_task` → before `organization_run_task_global_settings`, `workspace_run_task`.
- `registry_provider`/`registry_gpg_key` — independent.
- **data sources** → each after its backing resource (its fixture creates the resource, then reads it).

## Resource waves (respect deps; within a wave, parallel-safe)

**Wave 1 — roots (no intra-v0.1 deps):**
`tfe_project` · `tfe_workspace` · `tfe_team` · `tfe_agent_pool` · `tfe_variable_set` ·
`tfe_organization_membership` · `tfe_organization_default_settings` · `tfe_terraform_version` ·
`tfe_organization_token` · `tfe_audit_trail_token` · `tfe_notification_configuration` ·
`tfe_registry_provider` · `tfe_registry_gpg_key` · `tfe_organization_run_task`

**Wave 2 — depend on Wave 1:**
`tfe_project_settings` · `tfe_workspace_settings` · `tfe_variable` (workspace/varset scope) ·
`tfe_workspace_variable_set` · `tfe_project_variable_set` · `tfe_team_access` ·
`tfe_team_project_access` · `tfe_team_members` · `tfe_team_organization_member` ·
`tfe_team_organization_members` · `tfe_team_token` · `tfe_team_notification_configuration` ·
`tfe_agent_pool_allowed_workspaces` · `tfe_agent_pool_allowed_projects` ·
`tfe_agent_pool_excluded_workspaces` · `tfe_agent_token` ·
`tfe_project_notification_configuration` · `tfe_organization_run_task_global_settings` ·
`tfe_workspace_run_task` · `tfe_run_trigger` · `tfe_workspace_run` ·
`tfe_azure_oidc_configuration` · `tfe_aws_oidc_configuration` · `tfe_gcp_oidc_configuration` ·
`tfe_vault_oidc_configuration`

**Wave 3 — data sources (each after its backing resource is green):**
all 22, in two sub-batches so a data source never runs before its resource:
- after Wave 1: `tfe_project` · `tfe_projects` · `tfe_workspace` · `tfe_workspace_ids` · `tfe_team` ·
  `tfe_teams` · `tfe_agent_pool` · `tfe_variable_set` · `tfe_organization_members` ·
  `tfe_organization_membership` · `tfe_current_user` · `tfe_organization_run_task` ·
  `tfe_registry_provider` · `tfe_registry_gpg_key` · `tfe_registry_providers` ·
  `tfe_registry_gpg_keys`
- after Wave 2: `tfe_team_access` · `tfe_team_project_access` · `tfe_variables` · `tfe_outputs` ·
  `tfe_organization_run_task_global_settings` · `tfe_workspace_run_task`

## Per-task contract

For each resource/data-source task:
- **Register** both names in the mux (append to the seam from T0 if not already batched there).
- **Fixture** generated from that spec's acceptance criteria (`plan.md` §5), under `test/fixtures/`.
- **Acceptance** via `dev_overrides` against the dev stack — create → no-drift → destroy → 404 + every
  spec criterion. Loop-until-green (bounded, e.g. 3 attempts); on persistent failure mark the task
  `failed` with the diff and **stop** — never mark done.
- **Runtime** criterion exercised where the spec defines one (a run authenticates, a trigger fires, a
  token registers an agent). Note any real-dependency loop that stays a manual/staging check.
- **Docs** page (`docs/resources|data-sources/<name>.md`) + example (`examples/...`) from the spec.
- **PR** `feat(provider): <name>` linking the spec + sub-issue; auto-merge-on-green per the bootstrap
  decision.

## Native waves (after forked v0.1 — real codegen on `internal/stackweaver`)

The 17 native resources + 9 data sources depend on the native client, so they follow the forked waves.
Each native task additionally writes the resource's Go code (schema + CRUD via the native client), not
just registration. Envelope (JSON:API vs plain-JSON) per the resource's spec.

**T-native-0 — native client core:** finish `internal/stackweaver` (both codecs: JSON:API + plain
JSON; auth; pagination; error mapping). Blocks all native tasks.

**Native Wave A — Ansible core (roots):** `ansible_playbook` · `ansible_inventory` · `ansible_credential` ·
`ansible_config` · `ansible_notification_template`. Then depending on those: `ansible_host` ·
`ansible_group` · `ansible_inventory_source` (needs inventory + a cloud credential).

**Native Wave B — job templates:** `ansible_job_template` (needs playbook + inventory) →
`ansible_job_template_variable` · `ansible_job_template_credential` (needs template + credential) ·
`ansible_notification_attachment` (needs template + notification-template) · `ansible_schedule`
(needs a target) · `ansible_job` trigger (needs a job template; ships with only `extra_vars` overrides
until the backend gap closes).

**Native Wave C — workflows (DAG): DEFERRED.** `ansible_workflow` / `ansible_workflow_node` /
`ansible_workflow_edge` are spec'd but NOT built — the Ansible workflow engine is unused and unverified
(likely non-functional). Excluded until the engine is proven working (backing not green).

**Native Wave D — data sources:** the 9 native data sources, each after any resource its fixture
creates (runners/webhook-events/collections/adhoc-modules need no backing resource; the VCS + inventory-
sync ones read an existing connection/inventory).

Per-native-task contract = the same as the forked per-task contract **plus** writing the resource's Go
implementation against the native client (the forked tasks skip that — their code is upstream's).

## Explicitly out of v0.1 (tracked, not built)

`tfe_organization`, `tfe_team_member`, `tfe_github_app_installation`, `tfe_registry_module`
(partial/blocked — backing API not green), and the whole `dropped` set. The native surface (Ansible,
runners, VCS connections, …) is Phase 5 via the same `spec → plan → tasks → implement` path.

## Coverage check (no silent truncation)

Forked target = **39 resources + 22 data sources = 61 tasks** + T0. Native target = **14 resources (of 17 spec'd; 3 workflow deferred) +
9 data sources = 23 tasks** + T-native-0. Full spec'd surface = **56 resources + 31 data sources**; build scope excludes the 3 deferred workflow resources. If a task
is skipped or a wave trimmed, it is recorded here and reported — a green run must account for every
task or name what it left.
