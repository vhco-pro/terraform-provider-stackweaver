<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_inventory_source
tfe_alias: n/a
kind: resource
family: ansible
origin: native
backing_api: implemented
client_approach: native-client
status: spec'd
upstream_file: n/a (native — no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + core/models/ansible_inventory_source.go)
---
# stackweaver_ansible_inventory_source

**Native resource — no TFE equivalent.** Attaches a dynamic inventory source to an existing Ansible
inventory: hosts are discovered from a cloud provider (AWS EC2, Azure VMs, GCP Compute, VMware
vCenter) or a custom script, using an optional cloud credential and provider-specific `config`. A
sync action (immediate or scheduled) runs `ansible-inventory` and populates the inventory's hosts and
groups. Model: `core/models/ansible_inventory_source.go` (`AnsibleInventorySource`).

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `AnsibleInventorySources`
service (List/Create/Read/Update/Delete + a `Sync` action) calling the Stackweaver Ansible API over
HTTP. Unlike the plain-`json` playbook API, this endpoint uses a **JSON:API-shaped envelope**
(`{"data":{"type","attributes","relationships"}}`) with **kebab-case** attribute keys — see the
handler `formatInventorySourceResponse` in
`backend/internal/api/v2/handlers/ansible/inventory_sources.go`. The native client marshals that
envelope (confirm exact keys against the handler at implement time).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string (uuid) | Computed | — | — | no | server-assigned |
| `inventory_id` | string (uuid) | Required | yes | — | no | owning inventory; set only via the create path `/ansible/inventories/:id/sources`, never updatable |
| `name` | string | Required | no | — | no | 1–255 chars |
| `description` | string | Optional | no | `""` | no | |
| `source_type` | string | Required | yes | — | no | `aws` \| `azure` \| `gcp` \| `vmware` \| `custom`; not mapped by the Update handler → effectively immutable |
| `credential_id` | string (uuid) | Optional | no | — | no | cloud credential; **written as the `credential-id` attribute, read back as `relationships.credential.data.id`** (asymmetric). Empty string on update clears the credential (switch to OIDC) |
| `config` | string (JSON) | Optional+Computed | no | `{}` | no | provider-specific JSONB (AWS regions/filters, Azure resource_groups, GCP projects/zones, VMware hostname…); model as a `jsonencode`d string, normalize to avoid key-order drift |
| `update_on_launch` | bool | Optional+Computed | no | `true` | no | sync before each job run |
| `update_cache_timeout` | int | Optional+Computed | no | `0` | no | seconds a prior sync stays fresh for update-on-launch (0 = always sync) |
| `overwrite` | bool | Optional+Computed | no | `false` | no | remove source-owned hosts/groups the provider no longer reports (AWX parity) |
| `overwrite_vars` | bool | Optional+Computed | no | `false` | no | replace host vars wholesale on sync (false = merge) |
| `verbosity` | int | Optional+Computed | no | `0` | no | 0–4, adds `-v..-vvvv` to `ansible-inventory` |
| `group_by_instance_id` | bool | Optional+Computed | no | `false` | no | |
| `group_by_region` | bool | Optional+Computed | no | `true` | no | |
| `group_by_availability_zone` | bool | Optional+Computed | no | `false` | no | |
| `group_by_tag` | string | Optional | no | `""` | no | tag key to group by (e.g. `Environment`) |
| `hostname_var` | string | Optional+Computed | no | `public_ip` | no | which var becomes the ansible hostname |
| `instance_filters` | string | Optional | no | `""` | no | JSON array of filters |
| `sync_schedule` | string | Optional | no | `""` | no | cron expression for scheduled sync (e.g. `0 */6 * * *`) |
| `enabled` | bool | Optional+Computed | no | `true` | no | disabled sources reject sync |
| `status` | string | Computed | — | — | no | `never_synced` \| `syncing` \| `successful` \| `failed` |
| `last_sync_at` / `last_sync_error` / `last_sync_log` / `hosts_count` | (various) | Computed | — | — | no | sync-status readback |

## Wire contract

- **Create:** `POST /ansible/inventories/:id/sources` — `inventory_id` comes from the **path, not the
  body**. Body attributes: `name`, `description?`, `source-type`, `credential-id?`, `config?`,
  `sync-schedule?`, `update-on-launch?`, `update-cache-timeout?`, `overwrite?`, `overwrite-vars?`,
  `verbosity?`, `group-by-instance-id?`, `group-by-region?`, `group-by-availability-zone?`,
  `group-by-tag?`, `hostname-var?`, `instance-filters?`, `enabled?`. (Handler creates the source then
  applies the sync-behavior attributes in a second internal update.) → `201`, `type: inventory-sources`.
- **Read:** `GET /ansible/inventory-sources/:source_id`.
- **Update:** `PATCH /ansible/inventory-sources/:source_id` — name/description/config/enabled/grouping/
  hostname/filters/schedule/sync-behavior in place. `credential-id: ""` clears the credential.
  `source-type` is **not** applied by the update handler → treat as ForceNew. `inventory_id` cannot
  be changed → ForceNew.
- **Delete:** `DELETE /ansible/inventory-sources/:source_id` → `204`.
- **Action (not part of CRUD):** `POST /ansible/inventory-sources/:source_id/actions/sync` — enqueues
  an `ansible_sync` job to the ansible-runner via Redis; returns `202 Accepted` with the source in
  `status: syncing`. Rejected with `400` if the source is disabled. Exposed as an optional action, not
  a lifecycle requirement.
- **JSON:API type:** `inventory-sources`. Note the **asymmetry**: `credential_id` is a request
  *attribute* but a response *relationship* (`relationships.credential.data.id`). `config` is JSONB —
  the server defaults it to `{}` and may reorder keys; normalize to avoid perpetual diff.

## Acceptance criteria (these ARE the test)

Concrete, testable. The `implement` pipeline generates the fixture assertions from these.

1. `apply` of `{inventory_id, name, source_type = "aws", config}` creates the source; `id`, `name`,
   `source_type`, `config` round-trip into state.
2. Re-`plan` after apply shows **no drift** — computed `status`/`last_sync_*`/`hosts_count` and the
   settled defaults must not cause a perpetual diff.
3. Omitted defaults settle server-side: `update_on_launch=true`, `group_by_region=true`,
   `hostname_var="public_ip"`, `verbosity=0`, `overwrite=false`, `overwrite_vars=false`,
   `update_cache_timeout=0`, `enabled=true`.
4. Updating `description`/`config`/`enabled`/grouping flags applies in place; changing `inventory_id`
   or `source_type` forces recreate.
5. Setting then clearing `credential_id` (to `""`) round-trips: after clear, the read response carries
   no `relationships.credential`.
6. `destroy` removes it; a subsequent `GET /ansible/inventory-sources/:source_id` returns 404.

## Runtime criterion

The source is a live discovery pointer, not dead config: the `sync` action enqueues an `ansible_sync`
job that runs `ansible-inventory` against the provider and populates the parent inventory's hosts and
groups (`hosts_count` moves off 0, `status` becomes `successful`). Verified: create a source against a
reachable provider (or a `custom` script fixture), trigger `sync`, and assert the inventory gains
hosts and `status` transitions `never_synced → syncing → successful`.

## Docs + example

- Provider docs page: `docs/resources/ansible_inventory_source.md` — document the `config` JSON keys
  per `source_type` (reference the `*InventoryConfig` structs in the model, do not copy them) and the
  optional `sync` action.
- Example: `examples/resources/stackweaver_ansible_inventory_source/resource.tf` — an inventory + an
  AWS source referencing a `stackweaver_ansible_credential` id, with a minimal `config`.

## Divergences from upstream / TFE

Native resource — no TFE equivalent. `sync` is a Stackweaver action with no Terraform analogue
(surfaced as an optional action, not a lifecycle requirement). `source_type` is accepted in the update
request struct but not applied by the handler, so it is modeled ForceNew rather than in-place. The
per-provider `config` payload is intentionally free-form JSON rather than typed schema blocks, matching
the JSONB column.
