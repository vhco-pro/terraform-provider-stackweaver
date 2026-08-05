<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_workflow_node
tfe_alias: n/a
kind: resource
family: ansible
origin: native
backing_api: unverified
client_approach: native-client
status: deferred
upstream_file: n/a (native — no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + core/models/ansible_workflow.go)
---
# stackweaver_ansible_workflow_node

> **DEFERRED (not in build scope).** The Ansible workflow engine is unused and unverified (likely non-functional); these resources stay spec'd but are not built until the engine is proven working. Backing API not green.

**Native resource — no TFE equivalent.** A single step (vertex) in an
`stackweaver_ansible_workflow` DAG. A node runs one of four things — a job template, a nested
workflow, an inventory sync, or a manual approval gate — with optional per-node run overrides
(extra vars, limit, tags, verbosity). Model: `core/models/ansible_workflow.go`
(`AnsibleWorkflowNode`). Edges between nodes are managed by `stackweaver_ansible_workflow_edge`.

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `AnsibleWorkflowNodes`
service (List/Create/Read/Update/Delete). Same JSON:API-shaped envelope as the workflow resource:
`{"data": {"type": "ansible-workflow-nodes", "attributes": {...dasherized...},
"relationships": {"job-template"|"inventory"|"credential": {"data": {"id"}}}}}`. `extra-vars` is a
JSON **string** on the wire. Note there is **no dedicated Read-one endpoint**: the client reads a
node by `GET /ansible/workflows/:workflow_id/nodes` and selecting the node whose `id` matches (the
node's own `workflow_id` is needed for the list call — persist it in state).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string (uuid) | Computed | — | — | no | server-assigned |
| `workflow_id` | string (uuid) | Required | yes | — | no | owning workflow; POST path `/ansible/workflows/:id/nodes`. Needed for Read (list) |
| `node_type` | string | Required | yes | `job_template` | no | `job_template` \| `workflow` \| `inventory_sync` \| `approval`. Update ignores it → ForceNew |
| `identifier` | string | Required | yes | — | no | unique within the workflow. Update ignores it → ForceNew |
| `job_template_id` | string (uuid) | Optional | yes | — | no | `relationships.job-template` — target for `job_template` nodes. Update ignores → ForceNew |
| `inventory_id` | string (uuid) | Optional | no | — | no | `relationships.inventory` — per-node inventory override. **Updatable** relationship? No — Update reads only attributes; the `inventory` relationship is set at create only → ForceNew |
| `credential_id` | string (uuid) | Optional | yes | — | no | `relationships.credential` — per-node credential override. Update ignores → ForceNew |
| `position_x` | number (float64) | Optional | no | `0` | no | visual-editor coordinate; updatable |
| `position_y` | number (float64) | Optional | no | `0` | no | visual-editor coordinate; updatable |
| `extra_vars` | string (JSON) | Optional | no | `"{}"` | no | JSON string on the wire; updatable |
| `limit` | string | Optional | no | `""` | no | per-node host limit; updatable |
| `tags` | string | Optional | no | `""` | no | `--tags`; updatable |
| `skip_tags` | string | Optional | no | `""` | no | `--skip-tags`; updatable |
| `verbosity` | int | Optional | no | `0` | no | ansible `-v` level; updatable |
| `all_parents_must_converge` | bool | Optional | no | `false` | no | wait for **all** incoming edges before running; updatable |
| `approval_timeout` | int | Optional | yes | `0` | no | seconds, `0` = no timeout (approval nodes). Update ignores → ForceNew |
| `approval_message` | string | Optional | yes | `""` | no | prompt shown at the approval gate. Update ignores → ForceNew |
| `created_at` | string (rfc3339) | Computed | — | — | no | timestamp |

## Wire contract

- **Create:** `POST /ansible/workflows/:id/nodes` — `data.attributes`: `node-type`, `identifier`,
  `position-x`, `position-y`, `extra-vars` (JSON string), `limit`, `tags`, `skip-tags`, `verbosity`,
  `all-parents-must-converge`, `approval-timeout`, `approval-message`; `data.relationships`:
  `job-template.data.id?`, `inventory.data.id?`, `credential.data.id?`. Returns `201` with the
  formatted node (type `ansible-workflow-nodes`).
- **Read:** `GET /ansible/workflows/:workflow_id/nodes` → array; select by `id`. (No `GET
  /ansible/workflow-nodes/:id` single-fetch route exists.)
- **Update:** `PATCH /ansible/workflow-nodes/:id` — **partial**: only `position-x`, `position-y`,
  `limit`, `tags`, `skip-tags`, `verbosity`, `all-parents-must-converge`, `extra-vars` are applied.
  `node-type`, `identifier`, `approval-timeout`, `approval-message`, and **all** relationships
  (`job-template`/`inventory`/`credential`) are ignored on update → every one of those must force
  recreate.
- **Delete:** `DELETE /ansible/workflow-nodes/:id` → `204`.
- **JSON:API type:** `ansible-workflow-nodes`. Read may add a computed `job-template-name` attribute
  when the job template is preloaded (informational; do not treat as settable).

## Acceptance criteria (these ARE the test)

Concrete, testable. The `implement` pipeline generates the fixture assertions from these.

1. `apply` of a `job_template` node (`{workflow_id, node_type="job_template", identifier,
   job_template_id}`) creates the node; `id`, `node_type`, `identifier`, `job_template_id`,
   `position_x`, `position_y` round-trip into state (read via the workflow's node list).
2. Re-`plan` after apply shows **no drift** (`extra_vars` `"{}"` readback must not perpetually diff;
   float `position_*` must round-trip exactly).
3. Updating `position_x`/`position_y`, `limit`, `tags`, `skip_tags`, `verbosity`,
   `all_parents_must_converge`, or `extra_vars` applies **in place**.
4. Changing `node_type`, `identifier`, `job_template_id`, `inventory_id`, `credential_id`,
   `approval_timeout`, or `approval_message` **recreates** (ForceNew — the PATCH path ignores them).
5. An `approval` node (`node_type="approval"`, `approval_message`, `approval_timeout`) creates and
   round-trips those fields; changing the message/timeout recreates.
6. `destroy` removes the node; a subsequent workflow node-list no longer contains its `id`.

## Runtime criterion

A node is a declarative vertex whose runtime effect appears when the parent workflow launches: the
engine instantiates a workflow-node-job per node, runs the referenced job template (or blocks at an
approval gate / performs an inventory sync) applying the per-node `extra_vars`/`limit`/`tags`/
`verbosity`/`inventory` overrides, and honors `all_parents_must_converge` for fan-in. Verified:
build a two-node workflow joined by an edge, launch it, and both nodes produce node-jobs that execute
in edge order with the overrides applied. Config-with-real-effect, not dead CRUD.

## Docs + example

- Provider docs page: `docs/resources/ansible_workflow_node.md`.
- Example: `examples/resources/stackweaver_ansible_workflow_node/resource.tf` — a workflow with two
  `job_template` nodes (distinct `identifier`s, `position_*` laid out) plus per-node `extra_vars`,
  and an `approval` node, cross-referenced by the edge example.

## Divergences from upstream / TFE

Native resource — no TFE analogue (AWX Workflow Job Template Node concept).

- **Partial update / broad ForceNew.** The PATCH handler applies only a subset of attributes; the
  provider must mark `node_type`, `identifier`, `approval_timeout`, `approval_message`, and all three
  target relationships (`job_template_id`, `inventory_id`, `credential_id`) as ForceNew because the
  API silently ignores changes to them.
- **`nested_workflow_id` and `inventory_source_id` are not wired.** The model carries
  `NestedWorkflowID` (for `node_type="workflow"`) and `InventorySourceID` (for
  `node_type="inventory_sync"`), but `CreateNodeRequest` exposes **no** relationship for either.
  Flag as **blocked**: `workflow` (nested) and `inventory_sync` node types cannot have their target
  set through this API today — either omit those attributes or document the gap. Only `job_template`
  and `approval` node types are fully expressible.
- **No single-node Read endpoint.** Read is via the workflow's `GET .../nodes` list; `workflow_id`
  must be persisted in state to perform the read and detect out-of-band deletion.
- **Envelope divergence.** JSON:API-shaped body with dasherized keys, same as the workflow resource
  (not the plain-JSON playbook envelope).
