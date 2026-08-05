<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_inventory
tfe_alias: n/a
kind: resource
family: ansible
origin: native
backing_api: implemented
client_approach: native-client
status: spec'd
upstream_file: n/a (native — no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + core/models/ansible_inventory.go)
---
# stackweaver_ansible_inventory

**Native resource — no TFE equivalent.** Manages an Ansible inventory: a named collection of hosts and
groups, either org-scoped or project-scoped, of one of four types (`static`, `dynamic`, `vcs`,
`constructed`). Hosts (`stackweaver_ansible_host`), groups (`stackweaver_ansible_group`), and dynamic
sources hang off it; a sync action refreshes dynamic/VCS/constructed inventories.
Model: `core/models/ansible_inventory.go` (`AnsibleInventory`, ~line 47).

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `AnsibleInventories` service
(List/Create/Read/Update/Delete + a `Sync` action) calling the Stackweaver Ansible API over HTTP.

**Envelope is a JSON:API-style wrapper, not the plain model JSON of `stackweaver_ansible_playbook`.**
Requests wrap attributes as `{"data":{"type":"...","attributes":{...},"relationships":{...}}}` and the
attribute keys are **mixed-case** (verified in `handlers/ansible/inventories.go`
`CreateInventoryRequest`/`UpdateInventoryRequest`): snake_case for most fields
(`name`, `description`, `source`, `variables`, `vcs_repository`, `vcs_branch`, `inventory_path`) but
**kebab-case** for the type/constructed group (`inventory-type`, `source-vars`, `constructed-limit`,
`constructed-cache-timeout`, `input-inventory-ids`). The model's `type` field is keyed `inventory-type`
in the attributes object (to avoid clashing with the JSON:API `data.type`). `project_id` and
`vcs_connection_id` are passed as **relationships**, not attributes. The native client owns this
marshalling; confirm exact keys against the handler at implement time.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string (uuid) | Computed | — | — | no | server-assigned |
| `organization` | string | Required | yes | — | no | org **name**; URL scoping segment (`/organizations/:name/...`); maps to model `organization_id` |
| `project_id` | string (uuid) | Optional | yes | — | no | `relationships.project`; null = org-scoped, set = project-scoped; `(organization,name)` unique |
| `name` | string | Required | no | — | no | not-null; unique within org; updatable in place |
| `description` | string | Optional | no | `""` | no | |
| `type` | string | Optional+Computed | **yes** | `static` | no | `static` \| `dynamic` \| `vcs` \| `constructed`; **not in update body → ForceNew** |
| `variables` | map(string→any) / jsonb | Optional+Computed | no | `{}` | no | global inventory variables; may carry secret values (not encrypted at rest) |
| `source` | string | Optional | no | `""` | no | deprecated for VCS inventories; plugin config / legacy VCS URL |
| `vcs_connection_id` | string (uuid) | Optional | no | — | no | `relationships.vcs_connection`; GitHub-App VCS connection |
| `vcs_repository` | string | Optional | no | `""` | no | `"owner/repo"` (type `vcs`) |
| `vcs_branch` | string | Optional+Computed | no | `main` | no | |
| `inventory_path` | string | Optional | no | `""` | no | path to inventory file within the repo |
| `source_vars` | string | Optional | no | `""` | no | constructed: YAML compose/groups/keyed_groups rules |
| `constructed_limit` | string | Optional | no | `""` | no | constructed: host limit expression |
| `constructed_cache_timeout` | int | Optional+Computed | no | `0` | no | constructed: rebuild cache TTL seconds (0 = always) |
| `input_inventory_ids` | list(string uuid) | Optional | no | `[]` | no | constructed: ordered input inventories (`input-inventory-ids`) |
| `last_sync_at` | string (RFC3339) | Computed | — | — | no | sync readback |
| `last_sync_status` | string | Computed | — | — | no | `syncing` \| `successful` \| `failed` |
| `last_sync_error` | string | Computed | — | — | no | |
| `last_sync_hosts_discovered` | int | Computed | — | — | no | |
| `last_sync_log` | string | Computed | — | — | no | ansible-inventory stderr/warnings |
| `created_at` / `updated_at` | string (RFC3339) | Computed | — | — | no | |

## Wire contract

- **Create:** `POST /organizations/:name/ansible/inventories` — envelope
  `{"data":{"attributes":{"name","description","inventory-type","source","variables","vcs_repository",
  "vcs_branch","inventory_path","source-vars","constructed-limit","constructed-cache-timeout",
  "input-inventory-ids"},"relationships":{"project":{"data":{"id"}},"vcs_connection":{"data":{"id"}}}}}`.
  `name` is `binding:"required"`. Org taken from the `:name` path segment.
- **Read:** `GET /ansible/inventories/:id`.
- **Update:** `PATCH /ansible/inventories/:id` — same envelope with **pointer** attributes (all optional);
  `inventory-type` is **absent** from the update body, so `type` cannot change in place (ForceNew).
- **Delete:** `DELETE /ansible/inventories/:id`.
- **Action (not CRUD):** `POST /ansible/inventories/:id/actions/sync` — refreshes dynamic/VCS/constructed
  inventory. Optional resource action, not a lifecycle requirement.
- **Exports (read-only, not resource state):** `GET /ansible/inventories/:id/ini`,
  `GET /ansible/inventories/:id/json` — rendered inventory formats (data-source / import aid only).
- **Envelope:** JSON:API-style wrapper with mixed snake/kebab attribute keys (see Client approach).
  Response envelope is `{"data":{...},"meta":{"pagination":{...}}}` for List. Native client owns marshalling.

## Acceptance criteria (these ARE the test)

1. `apply` of `{organization, name, type="static"}` creates the inventory; `id`, `organization`,
   `name`, `type`, `variables` round-trip into state.
2. Re-`plan` after apply shows **no drift** — the computed sync fields (`last_sync_*`) and
   `created_at`/`updated_at` must not produce a perpetual diff; `variables`/`vcs_branch`/`type`/
   `constructed_cache_timeout` settle to their defaults (`{}`/`main`/`static`/`0`) when omitted.
3. Updating `name`/`description`/`variables`/`vcs_*`/`inventory_path`/`source_vars`/
   `constructed_limit`/`constructed_cache_timeout` applies **in place**.
4. Changing `type`, `organization`, or `project_id` forces **recreate** (ForceNew).
5. A project-scoped apply (`project_id` set) lands under that project; an org-scoped apply
   (`project_id` unset) is null-scoped and both re-`plan` clean.
6. For `type="constructed"`, `input_inventory_ids` round-trips as an ordered set of inventory ids.
7. `destroy` removes it; a subsequent `GET /ansible/inventories/:id` returns 404.

## Runtime criterion

The inventory is a real execution target, not dead config: a job template / job resolves it and jobs
run against its hosts, and (for dynamic/VCS/constructed) the `sync` action populates hosts/groups from
the external source. Verified: create an inventory, add a host, launch (or dry-run) a job template
bound to it, and confirm the host resolves; for a dynamic/VCS inventory, `sync` discovers hosts
(`last_sync_status = successful`, `last_sync_hosts_discovered > 0`).

## Docs + example

- Provider docs page: `docs/resources/ansible_inventory.md` — cover the four types and which attributes
  each type consumes.
- Example: `examples/resources/stackweaver_ansible_inventory/resource.tf` — a static org-scoped
  inventory plus a VCS-backed inventory (referencing a VCS connection id).

## Divergences from upstream / TFE

Native resource — no TFE equivalent. Wire divergence vs `stackweaver_ansible_playbook`: this endpoint
uses a **JSON:API-style envelope with mixed snake/kebab attribute keys** (`inventory-type`,
`source-vars`, `constructed-*`, `input-inventory-ids`), and `project`/`vcs_connection` are
relationships. `sync` and the `ini`/`json` exports are Stackweaver actions with no Terraform analogue.
