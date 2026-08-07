<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_host
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
# stackweaver_ansible_host

**Native resource - no TFE equivalent.** Manages a single host within a
`stackweaver_ansible_inventory`: a named target with an optional distinct hostname/IP, SSH port,
per-host variables, and an enabled flag. Model: `core/models/ansible_inventory.go`
(`AnsibleInventoryHost`, ~line 106).

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `AnsibleHosts` service
(List/Create/Read/Update/Delete) calling the Stackweaver Ansible API over HTTP.

**Envelope is a JSON:API-style wrapper** (like `stackweaver_ansible_inventory`, not the plain model
JSON of `stackweaver_ansible_playbook`): `{"data":{"type":"...","attributes":{...}}}` with snake_case
attribute keys and **no relationships block** (verified in `handlers/ansible/hosts.go`
`CreateHostRequest`/`UpdateHostRequest`). The parent `inventory_id` is taken from the create URL path,
not the body. The native client owns marshalling; confirm exact keys against the handler at implement
time.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string (uuid) | Computed | - | - | no | server-assigned |
| `inventory_id` | string (uuid) | Required | **yes** | - | no | owning inventory; URL path segment on create; `(inventory_id,name)` unique |
| `name` | string | Required | no | - | no | not-null; unique within inventory; updatable in place |
| `description` | string | Optional | no | `""` | no | |
| `hostname` | string | Optional | no | `""` | no | actual hostname/IP if different from `name` |
| `port` | int | Optional+Computed | no | `22` | no | SSH port |
| `variables` | map(string→any) / jsonb | Optional+Computed | no | `{}` | no | host-specific variables; may carry secret values (not encrypted at rest) |
| `enabled` | bool | Optional+Computed | no | `true` | no | disabled hosts are skipped at run time |
| `source_id` | string (uuid) | Computed | - | - | no | dynamic-source owner (null = manual); server/sync-set, **not** in create body |
| `created_at` / `updated_at` | string (RFC3339) | Computed | - | - | no | |

## Wire contract

- **Create:** `POST /ansible/inventories/:id/hosts` - envelope
  `{"data":{"attributes":{"name","description","hostname","port","variables","enabled"}}}`.
  `name` is `binding:"required"`; `enabled` is a nullable pointer (omit ⇒ server default `true`).
  `inventory_id` comes from the `:id` path segment.
- **Read:** `GET /ansible/hosts/:id`.
- **Update:** `PATCH /ansible/hosts/:id` - same envelope with **pointer** attributes
  (`name`, `description`, `hostname`, `port`, `variables`, `enabled`), all optional.
- **Delete:** `DELETE /ansible/hosts/:id`.
- **Envelope:** JSON:API-style wrapper, snake_case attribute keys, no relationships. Native client
  owns marshalling.

## Acceptance criteria (these ARE the test)

1. `apply` of `{inventory_id, name}` creates the host; `id`, `inventory_id`, `name`, `port`,
   `enabled` round-trip into state.
2. Re-`plan` after apply shows **no drift** - `port`/`variables`/`enabled` settle to defaults
   (`22`/`{}`/`true`), and computed `source_id`/`created_at`/`updated_at` must not cause a perpetual diff.
3. Updating `name`/`description`/`hostname`/`port`/`variables`/`enabled` applies **in place**.
4. Changing `inventory_id` forces **recreate** (ForceNew).
5. `source_id` is server/sync-owned: a manually-created host reads back `source_id = null` and it never
   appears in the create request nor causes drift.
6. `destroy` removes it; a subsequent `GET /ansible/hosts/:id` returns 404.

## Runtime criterion

The host is a real execution target: a job launched against its inventory includes this host in the
generated inventory (unless `enabled = false`, in which case it is excluded), applying its `hostname`
and `variables`. Verified: create a host, launch (or dry-run) a job template bound to the parent
inventory, and confirm the host appears in the rendered inventory with its port/vars.

## Docs + example

- Provider docs page: `docs/resources/ansible_host.md`.
- Example: `examples/resources/stackweaver_ansible_host/resource.tf` - an inventory plus two hosts,
  one with a distinct `hostname` and custom `port`/`variables`.

## Divergences from upstream / TFE

Native resource - no TFE equivalent. `source_id` is a read-only sync-ownership marker (dynamic-source
provenance) with no Terraform analogue - never user-writable, hence Computed. Wire uses the
JSON:API-style envelope with snake_case keys and inventory scoping via the URL path.
