<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_workspace
tfe_alias: tfe_workspace
kind: resource
family: workspaces
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_workspace.go
go_tfe_type: Workspace (WorkspaceCreateOptions / WorkspaceUpdateOptions)
compat_doc: docs/internal/tfe-compatibility/resources/workspaces/tfe_workspace.md
---
# stackweaver_workspace

The flagship resource: a workspace is the unit that holds Terraform config, state, variables and runs.
It can be CLI/API-driven or VCS-driven, and carries execution, VCS, and tagging settings. Maps 1:1 onto
Stackweaver's workspace concept (`ws-`-prefixed id).

## Client approach

`go-tfe-clean`. The upstream resource is a legacy SDKv2 resource
(`internal/provider/resource_tfe_workspace.go:31`) driving `go-tfe`'s `Workspaces` service verbatim
(`Create`, `ReadByIDWithOptions`/`ReadByID`, `UpdateByID`, `SafeDeleteByID`/`DeleteByID`, plus
`AddTags`/`RemoveTags` and `AddRemoteStateConsumers`/`RemoveRemoteStateConsumers`). Stackweaver serves
the stock `workspaces` JSON:API shape unchanged
(`docs/internal/tfe-compatibility/resources/workspaces/tfe_workspace.md`), including the `vcs-repo`
attribute object, the `agent-pool` relationship, and the `setting-overwrites` object required by
`tfe_workspace_settings`; no wrapper.

## Schema

Top-level attributes (kebab-case wire names in Notes where they differ):

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | `ws-` primary id |
| `name` | string | Required | no | — | no | unique in org; rename changes URL, not ForceNew |
| `organization` | string | Optional+Computed | yes | provider default | no | org name |
| `description` | string | Optional+Computed | no | — | no | |
| `agent_pool_id` | string | Optional+Computed | no | — | no | **deprecated** (use `tfe_workspace_settings`); conflicts with `operations`; sets `setting-overwrites` |
| `execution_mode` | string | Optional+Computed | no | — | no | **deprecated**; `remote`/`local`/`agent`; conflicts with `operations`; sets `setting-overwrites` |
| `operations` | bool | Optional+Computed | no | — | no | **deprecated**; conflicts with `execution_mode`/`agent_pool_id`; reads back `true` |
| `allow_destroy_plan` | bool | Optional | no | `true` | no | |
| `auto_apply` | bool | Optional+Computed | no | — | no | |
| `auto_apply_run_trigger` | bool | Optional | no | `false` | no | |
| `file_triggers_enabled` | bool | Optional | no | `true` | no | |
| `global_remote_state` | bool | Optional+Computed | no | — | no | deprecated on this resource; use `tfe_workspace_settings` |
| `queue_all_runs` | bool | Optional | no | `true` (provider) | no | TFE API default is `false`; provider defaults `true` |
| `speculative_enabled` | bool | Optional | no | `true` | no | |
| `structured_run_output_enabled` | bool | Optional | no | `true` | no | |
| `assessments_enabled` | bool | Optional+Computed | no | — | no | drift/health assessments |
| `project_id` | string | Optional+Computed | no | — | no | `project` relationship |
| `source_name` | string | Optional | no | — | no | requires `source_url`; "Created via <name>" |
| `source_url` | string | Optional | no | — | no | requires `source_name`; must be http(s) |
| `tag_names` | set(string) | Optional+Computed | no | — | no | via `AddTags`/`RemoveTags`; lowercase alnum + `:_-`, begin/end alnum |
| `terraform_version` | string | Optional+Computed | no | — | no | exact or constraint |
| `trigger_prefixes` | list(string) | Optional | no | — | no | conflicts with `trigger_patterns` |
| `trigger_patterns` | list(string) | Optional | no | — | no | conflicts with `trigger_prefixes` |
| `working_directory` | string | Optional | no | `""` | no | |
| `vcs_repo` | block (0/1) | Optional | no | — | no | VCS-driven workflow; see block below |
| `force_delete` | bool | Optional | no | `false` | no | force (unsafe) delete on destroy |
| `resource_count` | int | Computed | — | — | no | returned `0` by Stackweaver |
| `inherits_project_auto_destroy` | bool | Computed | — | — | no | out of scope (see Divergences) |
| `effective_tags` | map(string) | Computed | — | — | no | returns `{}` — key-value tag maps out of scope |

VCS repo block (`vcs_repo { ... }`, `TypeList` MaxItems=1; sent as the `vcs-repo` attribute object,
`VCSRepoOptions` fields are `json:` not `jsonapi:`):

| Attribute | Type | Req/Opt | Wire (json) | Notes |
|-----------|------|---------|-------------|-------|
| `identifier` | string | Required | `identifier` | e.g. `owner/repo` |
| `branch` | string | Optional | `branch` | default resolves to `main` server-side |
| `ingress_submodules` | bool | Optional | `ingress-submodules` | default `false` |
| `oauth_token_id` | string | Optional | `oauth-token-id` | Stackweaver VCS connection UUID; conflicts with `github_app_installation_id`; `AtLeastOneOf` the two |
| `github_app_installation_id` | string | Optional | `github-app-installation-id` | conflicts with `oauth_token_id`; `AtLeastOneOf` the two |
| `tags_regex` | string | Optional | `tags-regex` | tag-triggered runs; conflicts with `trigger_patterns`/`trigger_prefixes` |

## Wire contract

- **Create:** `Workspaces.Create(org, WorkspaceCreateOptions)` → `POST /organizations/:org/workspaces`.
  Booleans are `*bool omitempty`, so only configured/known values are sent. `execution_mode` /
  `agent_pool_id` / `operations` each set `setting-overwrites` (`execution-mode` + `agent-pool` = true);
  when none is set the provider sends `setting-overwrites` explicitly `false/false` to defer to
  project/org defaults. `tag_names` go out as the `tags` **relationship**; `tags` (key-value map, out of
  scope here) would be `tag-bindings`. `project_id` goes out as the `project` relationship. `vcs-repo`
  is a nested attribute object.
- **Read:** `Workspaces.ReadByIDWithOptions(id, {Include:[effective-tag-bindings]})` →
  `GET /workspaces/:id?include=effective-tag-bindings`, with a fallback to `ReadByID`
  (`GET /workspaces/:id`) if the include is unsupported. Response must carry `agent-pool` as a
  **relationship**, `setting-overwrites`, and `vcs-repo` (with `oauth-token-id`,
  `github-app-installation-id`, `branch`, `ingress-submodules`, `tags-regex`) when VCS is configured.
- **Update:** `Workspaces.UpdateByID(id, WorkspaceUpdateOptions)` → `PATCH /workspaces/:id`. Same
  attribute + `vcs-repo` handling. `tag_names` diffs are applied out-of-band via `AddTags`/`RemoveTags`
  (`POST`/`DELETE /workspaces/:id/relationships/tags`). Removing the `vcs_repo` block calls
  `RemoveVCSConnectionByID`.
- **Delete:** default is **safe-delete** — `Workspaces.SafeDeleteByID(id)` →
  `POST /workspaces/:id/actions/safe-delete` (gated on `permissions.can-force-delete` being present in
  the response, which Stackweaver advertises as `true`). With `force_delete = true`,
  `Workspaces.DeleteByID(id)` → `DELETE /workspaces/:id`. Safe-delete refuses (409) a workspace that
  still manages resources.
- **JSON:API type:** `workspaces`. No sensitive/write-only fields on this resource. `resource_count`
  reads back `0`; `operations` reads back `true`.

## Acceptance criteria (these ARE the test)

1. `apply` of a CLI-only workspace `{name, organization, description, terraform_version,
   working_directory, auto_apply, queue_all_runs, assessments_enabled}` creates it; `id` (`ws-`),
   `name`, `description`, `terraform_version`, `working_directory`, and the booleans round-trip into
   state.
2. Re-`plan` after apply shows **no drift**, including the computed `effective_tags = {}` and
   `resource_count`.
3. VCS block round-trips: with `vcs_repo { identifier, branch, github_app_installation_id }`, read
   returns a `vcs_repo` with the same `identifier`/`branch`/`github_app_installation_id` and
   `ingress_submodules = false`; the alternate form `vcs_repo { identifier, oauth_token_id }` round-trips
   its `oauth_token_id`. Supplying both `oauth_token_id` and `github_app_installation_id` fails at plan
   (ConflictsWith); supplying neither fails (AtLeastOneOf).
4. Execution: `execution_mode = "agent"` requires `agent_pool_id` (plan-time
   `validateAgentExecution` error otherwise), and setting `agent_pool_id` without
   `execution_mode = "agent"` also fails at plan. When set, the create/update request marks
   `setting-overwrites` true so the workspace stops deferring to project/org defaults.
5. `tag_names = ["env:dev","team-a"]` round-trips as a set; adding/removing an entry and re-`apply`
   applies exactly that delta via `AddTags`/`RemoveTags`; an invalid tag (uppercase / bad chars) fails
   at plan.
6. `trigger_prefixes` and `trigger_patterns` are mutually exclusive (plan-time ConflictsWith); each
   round-trips as an ordered list when used alone.
7. `organization` is ForceNew — changing it recreates; `name` change is an in-place rename (not a
   recreate).
8. Default `destroy` of an **empty** workspace safe-deletes and a subsequent `ReadByID` returns 404; a
   workspace that still manages resources refuses destroy (409) unless `force_delete = true`, which then
   hard-deletes it.

## Runtime criterion

Not CRUD-only — a workspace is the execution surface. Verified: a VCS-driven workspace
(`identifier` + `github_app_installation_id`) receives a webhook/VCS push and queues a run against the
configured `branch`/`working_directory`, honoring `trigger_prefixes`/`trigger_patterns`; an agent-mode
workspace (via `execution_mode`/`agent_pool_id` or `tfe_workspace_settings`) dispatches its run to the
named agent pool; and a run resolves its execution settings down the
`workspace -> project -> organization` chain when the workspace defers.

## Docs + example

- Provider docs page: `docs/resources/workspace.md` — full argument reference, the `vcs_repo` block,
  the deprecation of `execution_mode`/`agent_pool_id`/`global_remote_state` in favor of
  `tfe_workspace_settings`, safe-delete vs `force_delete`, computed `id`/`resource_count`, and import by
  id or `<org>/<name>`.
- Example: `examples/resources/stackweaver_workspace/resource.tf` — a project-scoped workspace with a
  `vcs_repo` block and a companion CLI-only workspace.

## Divergences from upstream / TFE

No wire-shape or value divergence — drop-in with `tfe_workspace` for the implemented surface (client is
go-tfe-clean). The following upstream attributes are **out of scope** (present in the provider schema
but not backed by Stackweaver; they are coverage gaps, not wire divergences):
`auto_destroy_at`, `auto_destroy_activity_duration`, `inherits_project_auto_destroy` (auto-destroy);
`ssh_key_id` (SSH keys); `remote_state_consumer_ids` (deprecated in TFE; managed via
`tfe_workspace_settings`); the key-value `tags` map and `ignore_additional_tag_names` /
`ignore_additional_tags` (only list-form `tag_names` is implemented; `effective_tags` returns `{}`);
`html_url` and `hyok_enabled`. `resource_count` is returned as `0`.
