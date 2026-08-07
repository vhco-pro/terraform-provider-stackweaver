<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_inventory_syncs
tfe_alias: n/a
kind: data-source
family: ansible
origin: native
backing_api: implemented
client_approach: native-client
status: spec'd
upstream_file: n/a (native - no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + core/models/ansible_inventory_sync.go)
---
# stackweaver_ansible_inventory_syncs

**Native data source - no TFE equivalent.** A read-only history helper that lists the sync runs
(AWX's "inventory update jobs") for one Ansible inventory: a dynamic-source sync, a VCS file-inventory
sync, or a constructed-inventory build, newest first. Exposes status, trigger, host/group counts, and
timestamps per run for observability and drift checks. Model:
`core/models/ansible_inventory_sync.go` (`AnsibleInventorySync`); handler:
`backend/internal/api/v2/handlers/ansible/inventory_syncs.go`.

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `AnsibleInventorySyncs`
service with `ListByInventory(ctx, inventoryID, {limit, offset})` →
`GET /ansible/inventories/:id/syncs`, and an optional `Read(ctx, syncID)` →
`GET /ansible/inventory-syncs/:sync_id` for a single run **including** its captured `output`. The
list endpoint returns a JSON:API-shaped envelope (`{"data": [...], "meta": {"total"}}`) with
dash-cased attribute keys; the native client marshals accordingly. Read-only history - no
create/update/delete.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `inventory_id` | string (uuid) | Required | - | - | no | the inventory whose sync history to list |
| `id` | string | Computed | - | - | no | set to `inventory_id` |
| `syncs` | list(object) | Computed | - | - | no | sync runs, newest first |
| `syncs[].id` | string (uuid) | Computed | - | - | no | sync run id |
| `syncs[].status` | string | Computed | - | - | no | `pending` \| `running` \| `successful` \| `failed` |
| `syncs[].triggered_by` | string | Computed | - | - | no | `manual` \| `schedule` \| `launch` \| `workflow` \| `webhook` |
| `syncs[].hosts_discovered` | number | Computed | - | - | no | |
| `syncs[].groups_discovered` | number | Computed | - | - | no | |
| `syncs[].source_name` | string | Computed | - | - | no | present when the run is a dynamic-source sync |
| `syncs[].error` | string | Computed | - | - | no | failure detail, empty on success |
| `syncs[].started_at` / `syncs[].finished_at` / `syncs[].created_at` | string (rfc3339) | Computed | - | - | no | nullable until the run runs/finishes |

Note: the list endpoint omits the per-run captured `output` (large text); it is only returned by the
single-run detail endpoint. This list data source does not expose `output`.

## Wire contract

- **Read/lookup:** `AnsibleInventorySyncs.ListByInventory(ctx, inventoryID, {limit, offset})` →
  `GET /ansible/inventories/:id/syncs?limit=&offset=` (default limit 20). Paginate to cover the
  history; `meta.total` gives the count.
- **Create/Update/Delete:** n/a - read-only data source.
- **JSON:API type:** `inventory-syncs`. Attributes are dash-cased (`triggered-by`,
  `hosts-discovered`, `groups-discovered`, `started-at`, `finished-at`, `created-at`, `source-name`,
  `error`); relationships carry `inventory` (and `source` when present). Native client maps dash-cased
  keys to the snake_case schema above. Sync output is intentionally excluded from the list.
- **Auth:** requires read-Ansible on the inventory's org (sync output/host detail is sensitive); a
  non-existent inventory → 404, a caller without permission → 403.

## Acceptance criteria (these ARE the test)

Assert against known dev-stack state: an inventory with a dynamic source that has been synced at least
once (so its history is non-empty).

1. Reading `data.stackweaver_ansible_inventory_syncs` with the fixture's `inventory_id` returns a
   non-empty `syncs` list ordered newest-first.
2. The computed `id` equals the requested `inventory_id`.
3. The most recent run's `status` is one of the known lifecycle values (e.g. `successful`),
   `triggered_by` is a known trigger, and `hosts_discovered`/`groups_discovered` are populated
   integers.
4. Each element carries an `id` (uuid) and a `created_at` timestamp.
5. `output` is **not** present on any element (list omits captured output by design).
6. Re-`plan` after apply shows **no drift** (`syncs`, `id` are Computed-only).

## Runtime criterion

Read-only observability helper over past sync runs. No runtime side effect beyond the history read;
its value is surfacing sync status/counts (e.g. gating downstream config on the latest run being
`successful`, or alerting on `failed`).

## Docs + example

- Provider docs page: `docs/data-sources/ansible_inventory_syncs.md` - argument `inventory_id`;
  computed `syncs` (each `id`/`status`/`triggered_by`/`hosts_discovered`/`groups_discovered`/
  `source_name`/`error`/`started_at`/`finished_at`/`created_at`) and `id`.
- Example: `examples/data-sources/stackweaver_ansible_inventory_syncs/data-source.tf` - read an
  inventory's sync history and expose the latest run's status via an output.

## Divergences from upstream / TFE

Native data source - no TFE equivalent. JSON:API envelope with dash-cased keys (mapped to snake_case
schema). The list intentionally omits the large `output` text (available only on the single-run
detail endpoint, `GET /ansible/inventory-syncs/:sync_id`); a per-run detail data source is out of
scope for this spec.
