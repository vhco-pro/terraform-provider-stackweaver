<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_group
tfe_alias: n/a
kind: resource
family: ansible
origin: native
backing_api: implemented
client_approach: native-client
status: spec'd
upstream_file: n/a (native - no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + core/models/ansible_inventory.go)
---
# stackweaver_ansible_group

**Native resource - no TFE equivalent.** Manages a group within a `stackweaver_ansible_inventory`: a
named grouping of hosts with group-level variables, optionally nested under a parent group. Model:
`core/models/ansible_inventory.go` (`AnsibleInventoryGroup`, ~line 138).

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `AnsibleGroups` service
(List/Create/Read/Update/Delete) calling the Stackweaver Ansible API over HTTP.

**Envelope is a JSON:API-style wrapper** (like `stackweaver_ansible_inventory`, not the plain model
JSON of `stackweaver_ansible_playbook`): `{"data":{"type":"...","attributes":{...},"relationships":{
"parent":{"data":{"id"}}}}}` with snake_case attribute keys (verified in `handlers/ansible/groups.go`
`CreateGroupRequest`/`UpdateGroupRequest`). The parent `inventory_id` comes from the create URL path;
`parent_id` (nested-group link) is a **relationship**, not an attribute. The native client owns
marshalling; confirm exact keys against the handler at implement time.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string (uuid) | Computed | - | - | no | server-assigned |
| `inventory_id` | string (uuid) | Required | **yes** | - | no | owning inventory; URL path segment on create; `(inventory_id,name)` unique |
| `name` | string | Required | no | - | no | not-null; unique within inventory; updatable in place |
| `description` | string | Optional | no | `""` | no | |
| `variables` | map(string→any) / jsonb | Optional+Computed | no | `{}` | no | group-specific variables; may carry secret values (not encrypted at rest) |
| `parent_id` | string (uuid) | Optional | no | - | no | `relationships.parent`; nested-group parent; updatable in place |
| `source_id` | string (uuid) | Computed | - | - | no | dynamic-source owner (null = manual); server/sync-set, **not** in create body |
| `created_at` / `updated_at` | string (RFC3339) | Computed | - | - | no | |

## Wire contract

- **Create:** `POST /ansible/inventories/:id/groups` - envelope
  `{"data":{"attributes":{"name","description","variables"},"relationships":{"parent":{"data":{"id"}}}}}`.
  `name` is `binding:"required"`. `inventory_id` comes from the `:id` path segment; `parent` relationship
  is optional (omit for a top-level group).
- **Read:** `GET /ansible/groups/:id`.
- **Update:** `PATCH /ansible/groups/:id` - same envelope with **pointer** attributes
  (`name`, `description`, `variables`) plus the optional `parent` relationship; all optional.
- **Delete:** `DELETE /ansible/groups/:id`.
- **Envelope:** JSON:API-style wrapper, snake_case attribute keys, `parent` as a relationship. Native
  client owns marshalling.

## Acceptance criteria (these ARE the test)

1. `apply` of `{inventory_id, name}` creates the group; `id`, `inventory_id`, `name`, `variables`
   round-trip into state.
2. Re-`plan` after apply shows **no drift** - `variables` settles to `{}`, and computed
   `source_id`/`created_at`/`updated_at` must not cause a perpetual diff.
3. Updating `name`/`description`/`variables`/`parent_id` applies **in place**.
4. Changing `inventory_id` forces **recreate** (ForceNew).
5. Nested groups round-trip: a child group with `parent_id` set to a sibling group's `id` reads back
   that parent and re-`plan`s clean; clearing `parent_id` promotes it to top-level in place.
6. `source_id` is server/sync-owned: a manually-created group reads back `source_id = null`, never
   appears in the create request, and never causes drift.
7. `destroy` removes it; a subsequent `GET /ansible/groups/:id` returns 404.

## Runtime criterion

The group shapes the rendered inventory a job runs against: hosts assigned to the group inherit its
`variables`, and nested groups compose their parents. Verified: create a group with a var, assign a
host to it, launch (or dry-run) a job template bound to the parent inventory, and confirm the group
and its group-vars appear in the rendered inventory.

## Docs + example

- Provider docs page: `docs/resources/ansible_group.md`.
- Example: `examples/resources/stackweaver_ansible_group/resource.tf` - an inventory, a parent group,
  and a child group referencing it via `parent_id`.

## Divergences from upstream / TFE

Native resource - no TFE equivalent. Host↔group membership (the `ansible_inventory_host_groups`
many-to-many) is **not** managed by this resource - it is out of scope here (no membership endpoint on
the group CRUD routes; assignment happens elsewhere). `source_id` is a read-only sync-ownership marker
with no Terraform analogue. Wire uses the JSON:API-style envelope with `parent` as a relationship.
