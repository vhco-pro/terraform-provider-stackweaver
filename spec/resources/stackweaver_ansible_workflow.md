<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_workflow
tfe_alias: n/a
kind: resource
family: ansible
origin: native
backing_api: unverified
client_approach: native-client
status: deferred
upstream_file: n/a (native - no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + core/models/ansible_workflow.go)
---
# stackweaver_ansible_workflow

> **DEFERRED (not in build scope).** The Ansible workflow engine is unused and unverified (likely non-functional); these resources stay spec'd but are not built until the engine is proven working. Backing API not green.

**Native resource - no TFE equivalent.** Registers an AWX-style workflow template: a named,
project-scoped orchestration container that stitches multiple job templates (and approval /
inventory-sync / nested-workflow steps) into a DAG. This resource manages the workflow *container*
and its launch-time defaults; the DAG itself is built from `stackweaver_ansible_workflow_node` and
`stackweaver_ansible_workflow_edge` resources that reference it. Model:
`core/models/ansible_workflow.go` (`AnsibleWorkflow`).

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `AnsibleWorkflows` service
(List/Create/Read/Update/Delete) calling the Stackweaver Ansible API over HTTP.

Unlike `stackweaver_ansible_playbook` (plain-`json` bodies), the workflow handler speaks a
**JSON:API-shaped envelope**: `{"data": {"type": "ansible-workflows", "id", "attributes": {...},
"relationships": {...}}}` with **dasherized** attribute keys (`allow-simultaneous`,
`ask-variables-on-launch`, `extra-vars`, …) and relationship objects for `project` / `inventory` /
`organization`. The native client must marshal this shape (confirm exact keys against
`handlers/ansible/workflows.go` at implement time). `extra-vars` crosses the wire as a **JSON string**,
not a nested object.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string (uuid) | Computed | - | - | no | server-assigned |
| `organization` | string (name) | Required | yes | - | no | org **name** in the POST path `/organizations/:name/...`; readback is the `organization` relationship (uuid) |
| `project_id` | string (uuid) | Optional+Computed | yes | org default project | no | `relationships.project.data.id`; when omitted the server falls back to the org's `default` project. Update ignores it → treat as ForceNew |
| `name` | string | Required | no | - | no | updatable in place |
| `description` | string | Optional | no | `""` | no | updatable in place |
| `allow_simultaneous` | bool | Optional | no | `false` | no | allow multiple concurrent runs |
| `ask_variables_on_launch` | bool | Optional | no | `false` | no | prompt for extra vars at launch |
| `ask_inventory_on_launch` | bool | Optional | no | `false` | no | prompt for inventory at launch |
| `ask_limit_on_launch` | bool | Optional | no | `false` | no | prompt for limit at launch |
| `inventory_id` | string (uuid) | Optional | yes | - | no | `relationships.inventory.data.id`; default inventory for runs. Update ignores it → treat as ForceNew |
| `extra_vars` | string (JSON) | Optional | no | `"{}"` | no | JSON-encoded string on the wire; echoed as a string |
| `limit` | string | Optional | no | `""` | no | default host limit for runs |
| `survey_enabled` | bool | Optional | no | `false` | no | updatable in place |
| `survey_spec` | string (JSON) | - | - | - | no | **not wired** - see divergences; present on the model but absent from the create/update request struct |
| `created_at` / `updated_at` | string (rfc3339) | Computed | - | - | no | timestamps |

## Wire contract

- **Create:** `POST /organizations/:name/ansible/workflows` - `data.attributes`: `name`,
  `description`, `allow-simultaneous`, `ask-variables-on-launch`, `ask-inventory-on-launch`,
  `ask-limit-on-launch`, `extra-vars` (JSON string), `limit`, `survey-enabled`;
  `data.relationships`: `project.data.id?`, `inventory.data.id?`. Returns `201` with the formatted
  workflow (type `ansible-workflows`).
- **Read:** `GET /ansible/workflows/:id` - returns the workflow plus `relationships.nodes` and
  `relationships.edges` collections (the resource itself only consumes the workflow attributes /
  scalar relationships; nodes and edges are owned by their own resources).
- **Update:** `PATCH /ansible/workflows/:id` - same attribute set as create. **Only** `name`,
  `description`, the four `ask-*`/`allow-*` flags, `extra-vars`, `limit`, `survey-enabled` are
  applied; `project` and `inventory` relationships are **not** re-read on update, so changes to
  `project_id` / `inventory_id` must force recreate.
- **Delete:** `DELETE /ansible/workflows/:id` → `204`. Cascades to nodes (`OnDelete:CASCADE`).
- **JSON:API type:** `ansible-workflows`. `extra-vars` is a JSON string (never a nested object).
  `survey-spec` is never accepted or echoed by these handlers.

## Acceptance criteria (these ARE the test)

Concrete, testable. The `implement` pipeline generates the fixture assertions from these.

1. `apply` of `{organization, name}` (optionally `project_id`, `inventory_id`, flags) creates the
   workflow; `id`, `name`, `allow_simultaneous`, `ask_*_on_launch`, `limit`, `survey_enabled`
   round-trip into state.
2. Re-`plan` after apply shows **no drift** (in particular `extra_vars` must normalize - a `"{}"`
   readback must not perpetually diff against an unset/empty config value).
3. `extra_vars` set to a JSON object string (e.g. `"{\"env\":\"prod\"}"`) round-trips byte-for-byte
   as a string.
4. Updating `name` / `description` / any `ask_*`/`allow_simultaneous` flag / `limit` /
   `survey_enabled` applies **in place** (no recreate).
5. Changing `project_id` or `inventory_id` **recreates** (ForceNew - the PATCH path ignores them).
6. `destroy` removes it; a subsequent `GET /ansible/workflows/:id` returns 404.
7. When `project_id` is omitted, state settles to the org's `default` project id with no perpetual
   diff.

## Runtime criterion

The workflow is the launch surface for a DAG of job templates: its runtime effect is that
`POST /ansible/workflows/:id/launch` starts a workflow job that walks the nodes/edges honoring
`allow_simultaneous`, the `ask_*_on_launch` prompts, `inventory_id`, `extra_vars`, and `limit`
defaults. Verified: create a workflow with at least one node + edge, launch it (or dry-run), and a
workflow job is created that resolves the configured defaults. Config-with-real-effect, not dead CRUD.

## Docs + example

- Provider docs page: `docs/resources/ansible_workflow.md`.
- Example: `examples/resources/stackweaver_ansible_workflow/resource.tf` - an org + project + a
  workflow with `ask_variables_on_launch` and a default `inventory_id`, cross-referenced by node/edge
  examples.

## Divergences from upstream / TFE

Native resource - no TFE analogue (AWX Workflow Job Template concept).

- **Ephemeral sub-routes are out of scope for this resource.** `POST /ansible/workflows/:id/launch`
  and `GET /ansible/workflows/:id/jobs` (and the `workflow-jobs` / `workflow-node-jobs` approval
  routes) are run-time execution/approval actions, not declarative config - they are **not** modeled
  here (a launch belongs to an ephemeral/action resource, if ever, à la `tfe_workspace_run`).
- **`survey_spec` is not wired.** The model carries `SurveySpec datatypes.JSON`, but
  `CreateWorkflowRequest` exposes only `survey-enabled` - there is no `survey-spec` field in the
  create/update payload and the formatter never echoes it. Treat as **blocked** at spec time: either
  omit the attribute or mark it computed-null and flag to the user that survey questions cannot be
  managed through this API surface today.
- **Envelope divergence.** Unlike the plain-JSON `stackweaver_ansible_playbook`, this surface uses a
  JSON:API-shaped body with dasherized keys and relationship objects; the native client owns this
  marshalling.
- `organization` is addressed by **name** (URL path), while the readback relationship is the org
  **uuid** - the client must reconcile the two.
