<!--
Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0
This is the spec matrix INDEX for terraform-provider-stackweaver. It is generated/maintained by the
`/provider-fork spec` pass and reviewed before any implementation. Per-resource formal specs live in
sibling files (spec/<name>.md); this file classifies the surface and records the client-fork verdict.
-->
# terraform-provider-stackweaver — spec matrix

Spec-first index of the provider surface. Every row is classified by **client approach**; the tally
of that column is the objective answer to "do we need to fork `go-tfe`?". Per-resource formal specs
(schema, wire contract, acceptance criteria) live in `spec/resources/<name>.md` and `spec/data-sources/<name>.md`. Nothing is implemented until
the spec is reviewed and approved.

- **Baseline:** upstream `terraform-provider-tfe` **v0.79.0** (the tracked watermark, and the latest
  upstream release as of 2026-08-05).
- **Repository strategy — clean split, not a mirror.** The repo has **its own git history** (a single
  initial commit at the v0.79.0 baseline), so it is a first-class Stackweaver provider rather than "a
  copy of the tfe provider with HashiCorp's history". `upstream` is a **fetch-only** remote; upstream
  changes are synced by **targeted diff of the supported files between release tags**, applied as
  native commits — **not** git merge. This is viable because our backport was always agent-driven
  per-file diff (not merge), and upstream churn is dominated by new resources we `drop` — the few
  changes to our supported resources are usually small/additive. MPL headers + this notice preserve
  attribution independent of git history.
- **Naming:** native `stackweaver_*` with `tfe_*` aliases (drop-in migration).
- **Derived from:** the compat matrix `docs/internal/tfe-compatibility/README.md` and its per-resource
  detail docs (the private source of divergence facts).

## Client-fork verdict

**No `go-tfe` fork. No wrappers. Dependency-only.** Of the 39 implemented resources, only 5 carry a
wire divergence, and **every one is value-level, not struct-shape** — stock `go-tfe` structs parse
them unchanged (bare-UUID ids that round-trip, email-as-id, an omitted optional relation, empty
metadata fields). None require a patched `go-tfe` call, so the `go-tfe + wrapper` count is **0**. Each
divergence is captured as a documented usage/migration note in its per-resource spec, not client code.

| Client approach | Resources | Data sources |
|---|---|---|
| `go-tfe (clean)` | 39 (5 with a documented value-level divergence note) | 22 (3 with a note) |
| `go-tfe + wrapper` | **0** | **0** |
| `native client` | 17 (the Stackweaver-native Ansible/AWX + runner surface) | 9 |

This verdict is provisional on each per-resource spec confirming its wire contract; the signal (0
struct-shape divergences across 39 resources) is overwhelming, but each `spec/<name>.md` re-checks it.

> **Counting note:** the compat matrix Quick-Status header says 37 implemented resources and its OIDC
> rollup row says 2, but the detailed per-family tables mark all **4** OIDC configs implemented — so
> the true count is **39**. This spec uses 39. `State Locking` is an implemented API surface, not a
> `tfe_*` resource, and is excluded.

## v0.1 scope

All 39 implemented resources + 22 implemented data sources. Partial/blocked rows (below) wait on their
backing API and are **not** in v0.1.

## Resources (39 — all `origin: forked`, `backing_api: implemented`, `client: go-tfe (clean)`)

| Resource (`tfe_*`) | Family | Wire divergence (→ documented note) | Status |
|---|---|---|---|
| `tfe_organization_membership` | Organizations | none | spec'd |
| `tfe_organization_default_settings` | Organizations | none | spec'd |
| `tfe_workspace` | Workspaces | none | spec'd |
| `tfe_workspace_settings` | Workspaces | none | spec'd |
| `tfe_team` | Teams | none (wire); **backing caveat** — `sso_team_id` + some org-access booleans accepted-but-not-enforced (see spec) | spec'd |
| `tfe_team_access` | Teams | none | spec'd |
| `tfe_team_project_access` | Teams | none | spec'd |
| `tfe_team_organization_member` | Teams | none | spec'd |
| `tfe_team_organization_members` | Teams | none | spec'd |
| `tfe_team_members` | Teams | **YES** — users relationship carries **email as `id`** (not username); `usernames` resolved by email | spec'd |
| `tfe_team_token` | Teams/Tokens | none (descriptioned BETA path out of scope) | spec'd |
| `tfe_team_notification_configuration` | Notifications | none | spec'd |
| `tfe_project` | Projects | none | spec'd |
| `tfe_project_settings` | Projects | none | spec'd |
| `tfe_variable` | Variables | none | spec'd |
| `tfe_variable_set` | Variables | none | spec'd |
| `tfe_workspace_variable_set` | Variables | none | spec'd |
| `tfe_project_variable_set` | Variables | **YES** — returns **bare UUID** project ids (not `prj-`), round-tripped verbatim; rejects attaching a project-owned set | spec'd |
| `tfe_run_trigger` | Runs | none | spec'd |
| `tfe_workspace_run` | Runs | none | spec'd |
| `tfe_agent_pool` | Agent Pools | none | spec'd |
| `tfe_agent_pool_allowed_workspaces` | Agent Pools | none | spec'd |
| `tfe_agent_pool_allowed_projects` | Agent Pools | none | spec'd |
| `tfe_agent_pool_excluded_workspaces` | Agent Pools | none | spec'd |
| `tfe_agent_token` | Agent Pools/Tokens | **YES** — `created-by` relation omitted (provider Read consumes `description` only) | spec'd |
| `tfe_azure_oidc_configuration` | OIDC | none | spec'd |
| `tfe_aws_oidc_configuration` | OIDC | none (token-from-file is runtime, not wire) | spec'd |
| `tfe_gcp_oidc_configuration` | OIDC | none (runtime delta only) | spec'd |
| `tfe_vault_oidc_configuration` | OIDC | none (wire field is `role`, matches go-tfe) | spec'd |
| `tfe_registry_provider` | Registry | **YES** — `registry-provider-versions` / `tag-bindings` relations omitted; v1 install returns package-metadata JSON, not a 302 | spec'd |
| `tfe_registry_gpg_key` | Registry | **YES** — `source` / `source-url` / `trust-signature` returned empty/default | spec'd |
| `tfe_organization_run_task` | Run Tasks | none (always-emitted `global-configuration` matches a go-tfe decode quirk) | spec'd |
| `tfe_organization_run_task_global_settings` | Run Tasks | none | spec'd |
| `tfe_workspace_run_task` | Run Tasks | none (`stage`+`stages` both emitted, wire-compat) | spec'd |
| `tfe_notification_configuration` | Notifications | none (email delivery deferred is behavioral, bytes round-trip) | spec'd |
| `tfe_project_notification_configuration` | Notifications | none | spec'd |
| `tfe_organization_token` | Tokens | none | spec'd |
| `tfe_audit_trail_token` | Tokens | none (`?token=audit-trails` is TFE's own dispatch) | spec'd |
| `tfe_terraform_version` | Versions | none (`deprecated-reason` nil-handling matches go-tfe `*string,omitempty`) | spec'd |

## Data sources (22 — `client: go-tfe (clean)`)

| Data source (`tfe_*`) | Wire divergence (→ note) | Status |
|---|---|---|
| `tfe_organization_members` | none | spec'd |
| `tfe_organization_membership` | none | spec'd |
| `tfe_current_user` | **YES** — `is_service_account` always `false` (no service-account kind yet) | spec'd |
| `tfe_workspace` | none | spec'd |
| `tfe_workspace_ids` | **YES** — `tag_names`/`exclude_tags` match on effective-tag **key** (any value) | spec'd |
| `tfe_project` | none | spec'd |
| `tfe_projects` | none | spec'd |
| `tfe_team` | **YES** — SCIM attrs (`scim_*`) not populated (Zitadel concern) | spec'd |
| `tfe_teams` | none | spec'd |
| `tfe_team_access` | none | spec'd |
| `tfe_team_project_access` | none | spec'd |
| `tfe_variable_set` | none | spec'd |
| `tfe_variables` | none | spec'd |
| `tfe_outputs` | none | spec'd |
| `tfe_agent_pool` | none | spec'd |
| `tfe_organization_run_task` | none | spec'd |
| `tfe_organization_run_task_global_settings` | none | spec'd |
| `tfe_workspace_run_task` | none | spec'd |
| `tfe_registry_provider` | none | spec'd |
| `tfe_registry_gpg_key` | none | spec'd |
| `tfe_registry_providers` | none | spec'd |
| `tfe_registry_gpg_keys` | none | spec'd |

## Partial / blocked (NOT in v0.1 — backing API not fully green)

| Resource | Backing API | Wire divergence | Disposition |
|---|---|---|---|
| `tfe_organization` | partial (~10 settings return defaults) | YES — non-JSON:API-shaped list response + hardcoded settings | `blocked` — route gaps to `/tfe-compat` before shipping |
| `tfe_team_member` | partial | YES — identifies member by org-membership id, not username | `blocked` |
| `tfe_registry_module` | partial (internal registry only) | none | `later` — CRUD-shippable, scope TBD |
| `tfe_github_app_installation` | partial (UI flow) | YES — different API surface (`/github-app/...`), constructed/omitted urls | `blocked` — likely stays a divergence |
| State Versions API (surface) | partial | YES — workspace-scoped only, `state_data` create body, `commit_hash` snake_case | not a `tfe_*` resource |

## Spec review notes (from the Phase 1 fan-out)

Surfaced while writing the per-resource specs; worth an eyeball before `implement`:

- **`tfe_team` backing caveat.** Wire shape is clean, but `sso_team_id` and the
  `manage_teams` / `manage_organization_access` / `access_secret_teams` org-access booleans are
  accepted-but-not-enforced by the backend (its spec is `backing_api: partial`). It still ships in
  v0.1 (CRUD + the enforced fields work); the unenforced fields are documented, not silently passed.
- **Two notification resources** (`tfe_notification_configuration`,
  `tfe_project_notification_configuration`) round-trip all bytes, but email delivery /
  `email_user_ids` / `delivery_responses` are behavioral deferrals (not wire divergences) — noted in
  their specs.
- **Cosmetic:** a few data-source specs set `compat_doc: n/a` (the sub-agent looked for the compat
  tree inside the provider repo rather than the monorepo). Source of record for those is the spec
  itself; harmless, normalize opportunistically.

## Native surface (`origin: native`, `client: native-client`)

Stackweaver-only resources with **no** TFE equivalent — mostly a full AWX-style Ansible subsystem plus
runner/VCS observability. **Spec'd (17 resources + 9 data sources); 3 workflow resources DEFERRED (engine unused/unverified) → 14 native resources in build scope.** Unlike the forked resources
these are real codegen against a new `internal/stackweaver` client. Built after the forked v0.1 waves
(they depend on the native client existing). See `plan/tasks.md` for the native waves.

**Envelope (VERIFY before implement):** the platform contract is JSON:API and most native endpoints
conform (`data.attributes`). The sub-agent envelope assessments proved **unreliable** — notifications
were mis-flagged as plain-JSON but are actually JSON:API (`type: ansible-notification-templates`,
`attributes`). So each spec's wire-contract envelope is **provisional** and must be re-verified per
endpoint at implement time. Confirmed genuine deviations from strict JSON:API:
- `ansible_config` returns `{data:{type, config_content}}` with attributes **flat** (not under
  `data.attributes`) — a real inconsistency worth normalizing (tracked in monorepo issue #608). Non-blocking for the provider.
- The **list/discovery helper** endpoints (VCS repositories/branches/yaml-files, playbook-file
  discovery, collections, adhoc-modules) return `{data:[...]}` of ad-hoc objects (repo names, file
  paths), not typed resources — fine as read-only data-source helpers (they back the UI pickers).

The native client handles both shapes, but the envelope column across the native specs needs a clean
per-endpoint audit before native implement (it must be right — fixtures depend on it).

### Native resources (17)

| Resource | Family | Notes / backing caveats |
|---|---|---|
| `stackweaver_ansible_playbook` | ansible | VCS-backed source pointer + `sync` action |
| `stackweaver_ansible_inventory` | ansible | static/dynamic/vcs/constructed; `type` ForceNew |
| `stackweaver_ansible_host` | ansible | static-inventory member; `source_id` computed |
| `stackweaver_ansible_group` | ansible | nested groups via `parent_id` |
| `stackweaver_ansible_inventory_source` | ansible | dynamic source; `source_type` immutable (update gap) |
| `stackweaver_ansible_credential` | ansible | 8 types; write-only secrets (only 4 `has_*` echoes); home of the parked `tfe_ssh_key`; `vault_id` not settable (API gap) |
| `stackweaver_ansible_config` | ansible | PUT-upsert singleton per org/project scope; workspace scope has no route |
| `stackweaver_ansible_job_template` | ansible | central AWX template; create/update asymmetries; `galaxy_requirements` not wired (API gap) |
| `stackweaver_ansible_job_template_variable` | ansible | template-scoped var; sensitive write-only; no single-row GET |
| `stackweaver_ansible_job_template_credential` | ansible | attach link (create/delete); synthesized composite id |
| `stackweaver_ansible_workflow` | ansible | **DEFERRED** — workflow engine unused/unverified (not built) |
| `stackweaver_ansible_workflow_node` | ansible | **DEFERRED** — with the workflow engine |
| `stackweaver_ansible_workflow_edge` | ansible | **DEFERRED** — with the workflow engine |
| `stackweaver_ansible_schedule` | ansible | cron; 4 target types |
| `stackweaver_ansible_notification_template` | ansible | webhook/email/teams; write-only `secret`; **no GET-by-id** → Read = list+filter |
| `stackweaver_ansible_notification_attachment` | ansible | attach link (create/delete) |
| `stackweaver_ansible_job` | ansible | **trigger** (mirrors `tfe_workspace_run`); `backing_api: partial` — `LaunchFromTemplate` accepts only `extra_vars` today, so limit/tags/inventory overrides need a small backend extension (route to `/tfe-compat`) |

### Native data sources (9)

| Data source | Notes |
|---|---|
| `stackweaver_runners` | self-hosted runner fleet + stats (runners self-register → no resource) |
| `stackweaver_vcs_repositories` / `_repository_branches` / `_yaml_files` | VCS browse/discovery (`_yaml_files` folds in inventory-files via `file_type`) |
| `stackweaver_ansible_vcs_playbook_files` | playbook discovery in a repo |
| `stackweaver_ansible_inventory_syncs` | inventory sync history |
| `stackweaver_ansible_collections` | pre-installed Galaxy collections (search is a backend stub) |
| `stackweaver_ansible_adhoc_modules` | effective org ad-hoc allowlist |
| `stackweaver_webhook_events` | VCS webhook delivery log |

### Native backing gaps (route to `/tfe-compat` before those parts ship)

Model fields present but not wired in the API today: `ansible_credential.vault_id`;
`job_template.galaxy_requirements`; `workflow.survey_spec`; `workflow_node` nested-workflow /
inventory-sync targets; `ansible_job` non-`extra_vars` launch overrides. Each is flagged in its spec;
the resource still ships without the unwired field.

### Overlaps handled in the forked spec (not native-new)

VCS Connection → the `tfe_github_app_installation` family (its listing endpoints became the native VCS
data sources above). Runner → `tfe_agent_pool` + `tfe_agent_token` (its listing became
`stackweaver_runners`). `tfe_ssh_key` → a later thin face over `stackweaver_ansible_credential`'s
`ssh` type.
