<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_workspace
tfe_alias: tfe_workspace
kind: data-source
family: workspaces
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_workspace.go
go_tfe_type: Workspace
compat_doc: docs/internal/tfe-compatibility/data-sources/workspaces/tfe_workspace.md
---
# stackweaver_workspace

Reads a single workspace by `name` within an organization and exposes its full settings plus its
`effective_tags` (own bindings merged with its project's inherited bindings, workspace-wins on a key
conflict). Maps onto Stackweaver's workspace record.

## Client approach

`go-tfe-clean`. The legacy-SDK data source calls
`Workspaces.ReadWithOptions(org, name, {Include: effective_tag_bindings})` →
`GET /organizations/:org/workspaces/:name?include=effective_tag_bindings` (go-tfe sends the include value
with underscores). If the instance rejects the include (`ErrInvalidIncludeValue`) it falls back to a
plain `Workspaces.Read`. Remote-state consumers are resolved via a secondary read. Stackweaver returns
the stock go-tfe `workspaces` JSON:API shape with the effective-tag-bindings relation
(`docs/internal/tfe-compatibility/data-sources/workspaces/tfe_workspace.md`); no wrapper.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `name` | string | Required | — | — | no | workspace name (lookup key) |
| `organization` | string | Optional | — | provider default | no | org name; falls back to the provider default |
| `id` | string | Computed | — | — | no | `workspaces` id (SetId) |
| `description` | string | Computed | — | — | no | |
| `effective_tags` | map(string) | Computed | — | — | no | own + project-inherited tag bindings (workspace wins) |
| `tag_names` | set(string) | Optional+Computed | — | — | no | legacy flat tag names; set from the read |
| `project_id` | string | Computed | — | — | no | owning project id (nil-safe if pre-projects instance) |
| `execution_mode` | string | Computed | — | — | no | |
| `terraform_version` | string | Computed | — | — | no | |
| `auto_apply`, `auto_apply_run_trigger`, `allow_destroy_plan`, `assessments_enabled`, `file_triggers_enabled`, `operations`, `queue_all_runs`, `speculative_enabled`, `structured_run_output_enabled`, `global_remote_state`, `project_remote_state`, `locked`, `hyok_enabled` | bool | Computed | — | — | no | workspace flags |
| `auto_destroy_at`, `auto_destroy_activity_duration`, `inherits_project_auto_destroy` | string/bool | Computed | — | — | no | auto-destroy settings (duration nullable on the wire) |
| `working_directory`, `source_name`, `source_url`, `source`, `environment`, `ssh_key_id`, `html_url` | string | Computed | — | — | no | |
| `trigger_prefixes`, `trigger_patterns` | list(string) | Computed | — | — | no | |
| `remote_state_consumer_ids` | set(string) | Computed | — | — | no | resolved via a secondary read when not globally/project shared |
| `resource_count`, `run_failures`, `runs_count`, `policy_check_failures`, `apply_duration_average`, `plan_duration_average` | int | Computed | — | — | no | counters/averages (averages in ms) |
| `vcs_repo` | list(object) | Computed | — | — | no | `{identifier, branch, ingress_submodules, oauth_token_id, tags_regex, github_app_installation_id}` |
| `setting_overwrites`, `permissions`, `actions` | map(bool) | Computed | — | — | no | inherited-setting flags / caller permissions / available actions |
| `created_at`, `updated_at` | string | Computed | — | — | no | RFC3339 |

## Wire contract

- **Read/lookup:** `Workspaces.ReadWithOptions(org, name, {Include: [effective_tag_bindings]})` →
  `GET /organizations/:org/workspaces/:name?include=effective_tag_bindings`; falls back to
  `Workspaces.Read` on `ErrInvalidIncludeValue`. Remote-state consumers via
  `readWorkspaceStateConsumers` when the workspace is neither globally nor project shared.
- **Create/Update/Delete:** n/a — read-only data source.
- **JSON:API type:** `workspaces`. `effective-tag-bindings` relation → `effective_tags` map;
  `auto-destroy-activity-duration` and `auto-destroy-at` are nullable on the wire. No divergence from
  stock go-tfe.

## Acceptance criteria (these ARE the test)

1. Fixture creates a `stackweaver_project` (tags `env`, `team`) and a `stackweaver_workspace` in it
   (tags `env=dev` override, `owner=me`), then reads `data.stackweaver_workspace` by `name`.
2. The data source's computed `id` equals the created workspace's id.
3. `effective_tags` round-trips the merged set (workspace-own merged with project-inherited; workspace
   wins on `env`), e.g. `{env=dev, team=platform, owner=me}`; `description` and `project_id` match the
   created workspace.
4. Re-`plan` after apply shows **no drift**.
5. **Quirk (do not assert before the read):** `tag_names` is Optional-not-Computed-only (Optional+Computed)
   and can be known-null at plan; assert the clearly-Computed fields (`id`, `effective_tags`,
   `description`, `project_id`) rather than input-shaped ones.

## Runtime criterion

Read-only data source. Resolves one workspace's settings and its effective (own + inherited) tags; no
runtime side effect beyond the read.

## Docs + example

- Provider docs page: `docs/data-sources/workspace.md` — arguments `name`/`organization`; the full
  computed attribute set (settings, `effective_tags`, `vcs_repo`, counters, `id`).
- Example: `examples/data-sources/stackweaver_workspace/data-source.tf` — read a workspace by name.

## Divergences from upstream / TFE

None. Drop-in with `tfe_workspace`; `effective_tags` depends on the project/workspace tags feature.
