<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_workflow_edge
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
# stackweaver_ansible_workflow_edge

> **DEFERRED (not in build scope).** The Ansible workflow engine is unused and unverified (likely non-functional); these resources stay spec'd but are not built until the engine is proven working. Backing API not green.

**Native resource — no TFE equivalent.** A directed connection (edge) between two
`stackweaver_ansible_workflow_node`s in a workflow DAG, carrying the condition under which the
target node runs after the source node finishes (`on_success` / `on_failure` / `always`). Model:
`core/models/ansible_workflow.go` (`AnsibleWorkflowEdge`). This is an **immutable link resource** —
it has no PATCH; any change replaces it (delete + recreate).

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `AnsibleWorkflowEdges`
service (List/Create/Delete — **no Update**). Same JSON:API-shaped envelope as the other workflow
resources: `{"data": {"type": "ansible-workflow-edges", "attributes": {"condition"},
"relationships": {"source-node": {"data": {"id"}}, "target-node": {"data": {"id"}}}}}`. As with
nodes, there is **no single-fetch Read route**: read by `GET /ansible/workflows/:workflow_id/edges`
and select the edge whose `id` matches (persist `workflow_id` in state for the read).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string (uuid) | Computed | — | — | no | server-assigned |
| `workflow_id` | string (uuid) | Required | yes | — | no | owning workflow; POST path `/ansible/workflows/:id/edges`. Needed for Read (list) |
| `source_node_id` | string (uuid) | Required | yes | — | no | `relationships.source-node.data.id` — edge origin |
| `target_node_id` | string (uuid) | Required | yes | — | no | `relationships.target-node.data.id` — edge destination |
| `condition` | string | Required | yes | `on_success` | no | `on_success` \| `on_failure` \| `always` |
| `created_at` | string (rfc3339) | Computed | — | — | no | timestamp |

**Every configurable attribute is ForceNew** — the resource is immutable (no PATCH endpoint).

## Wire contract

- **Create:** `POST /ansible/workflows/:id/edges` — `data.attributes.condition`;
  `data.relationships.source-node.data.id`, `data.relationships.target-node.data.id`. Returns `201`
  with the formatted edge (type `ansible-workflow-edges`).
- **Read:** `GET /ansible/workflows/:workflow_id/edges` → array; select by `id`. (No
  `GET /ansible/workflow-edges/:id` single-fetch route exists.)
- **Update:** **none** — there is no PATCH route. All attributes are ForceNew; any change is a
  delete + create.
- **Delete:** `DELETE /ansible/workflow-edges/:id` → `204`.
- **JSON:API type:** `ansible-workflow-edges`.

## Acceptance criteria (these ARE the test)

Concrete, testable. The `implement` pipeline generates the fixture assertions from these.

1. `apply` of `{workflow_id, source_node_id, target_node_id, condition="on_success"}` creates the
   edge; `id`, `source_node_id`, `target_node_id`, `condition` round-trip into state (read via the
   workflow's edge list).
2. Re-`plan` after apply shows **no drift**.
3. Changing `condition` (e.g. to `on_failure`) **recreates** the edge (ForceNew — no update path);
   the plan shows destroy + create, not an in-place update.
4. Changing `source_node_id` or `target_node_id` **recreates** the edge.
5. `destroy` removes the edge; a subsequent workflow edge-list no longer contains its `id`.
6. An edge with `condition="always"` and one with `condition="on_failure"` both create and
   round-trip their condition.

## Runtime criterion

An edge is a declarative DAG link whose runtime effect appears when the parent workflow launches: the
engine follows an edge from source to target only when the source node's outcome matches the edge
`condition` (`on_success` after success, `on_failure` after failure, `always` unconditionally),
gating whether/when the target node-job runs. Verified: build a two-node workflow with an
`on_success` edge, launch it, and the target node-job runs only after the source succeeds; an
`on_failure` edge's target is skipped on source success. Config-with-real-effect, not dead CRUD.

## Docs + example

- Provider docs page: `docs/resources/ansible_workflow_edge.md`.
- Example: `examples/resources/stackweaver_ansible_workflow_edge/resource.tf` — two nodes joined by
  an `on_success` edge (referencing `stackweaver_ansible_workflow_node.*.id`), demonstrating that
  editing `condition` triggers replacement.

## Divergences from upstream / TFE

Native resource — no TFE analogue (AWX Workflow Job Template Node "success/failure/always" links).

- **Immutable — no PATCH.** The API exposes only create/list/delete for edges; the provider models
  the whole resource as ForceNew (create = add link, delete = remove link). This mirrors the model's
  create-only fields (there is no `UpdatedAt`, only `CreatedAt`).
- **No single-edge Read endpoint.** Read is via the workflow's `GET .../edges` list; `workflow_id`
  must be persisted in state to perform the read and detect out-of-band deletion.
- **Envelope divergence.** JSON:API-shaped body with dasherized keys, same as the other workflow
  resources (not the plain-JSON playbook envelope).
