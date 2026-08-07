<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_collections
tfe_alias: n/a
kind: data-source
family: ansible
origin: native
backing_api: implemented
client_approach: native-client
status: spec'd
upstream_file: n/a (native - no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + backend/internal/api/v2/handlers/ansible/collections.go)
---
# stackweaver_ansible_collections

**Native data source - no TFE equivalent.** A read-only discovery helper that lists the Ansible
Galaxy collections **pre-installed** on the Stackweaver runner image, so configuration and playbooks
can assert a collection is available without shipping a `requirements.yml`. Handler:
`backend/internal/api/v2/handlers/ansible/collections.go` (`ListPreInstalledCollections`). The
pre-installed set mirrors `runner-images/ansible/Dockerfile`.

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `AnsibleCollections` service
with `ListPreInstalled(ctx)` → `GET /ansible/collections/pre-installed`. The endpoint returns a
JSON:API-shaped envelope (`{"data": [{"type":"ansible-collections","id":<name>,"attributes":{...}}]}`);
the native client marshals accordingly. Read-only/external - the collection set is a property of the
runner image, not a managed resource; no create/update/delete.

Optional (deferred): a `search` argument backed by `GET /ansible/collections/search?q=` - but that
endpoint is a not-yet-implemented placeholder (returns an empty `data` array with a "not yet
implemented" `meta.message`), so this spec models **pre-installed only** and treats search as
out-of-scope until the backend lands it.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | - | - | no | synthesized constant (e.g. `pre-installed`) |
| `collections` | list(object) | Computed | - | - | no | pre-installed collections on the runner |
| `collections[].name` | string | Computed | - | - | no | fully-qualified, e.g. `amazon.aws` (also the item id) |
| `collections[].namespace` | string | Computed | - | - | no | e.g. `amazon` |
| `collections[].version` | string | Computed | - | - | no | `latest` for pre-installed |
| `collections[].description` | string | Computed | - | - | no | |
| `collections[].source` | string | Computed | - | - | no | `pre-installed` for this data source |

## Wire contract

- **Read/lookup:** `AnsibleCollections.ListPreInstalled(ctx)` →
  `GET /ansible/collections/pre-installed`.
- **Create/Update/Delete:** n/a - read-only data source.
- **JSON:API type:** `ansible-collections`; item `id` is the collection name; attributes are
  `name`, `namespace`, `version`, `description`, `source`. Static server-side today (the set is
  hard-coded to match the runner image).
- **Out of scope:** `GET /ansible/collections/search` (placeholder, empty results) and
  `GET /ansible/jobs/:id/collections` (per-job listing, currently returns the same pre-installed set).

## Acceptance criteria (these ARE the test)

The pre-installed set is fixed dev-stack state (hard-coded in the handler), so assert exact members.

1. Reading `data.stackweaver_ansible_collections` returns a `collections` list containing the known
   pre-installed collections: `amazon.aws`, `azure.azcollection`, `google.cloud`, `community.vmware`,
   `community.general`, `ansible.posix`, `ansible.netcommon`.
2. For `amazon.aws`: `namespace` is `amazon`, `version` is `latest`, `source` is `pre-installed`.
3. The computed `id` is set (constant).
4. Re-`plan` after apply shows **no drift** (`collections`, `id` are Computed-only).

## Runtime criterion

Read-only discovery helper. Reports which Galaxy collections the runner image ships; no runtime side
effect. Its practical use is asserting/branching on collection availability before referencing a
collection's modules in a playbook.

## Docs + example

- Provider docs page: `docs/data-sources/ansible_collections.md` - computed `collections` (each
  `name`/`namespace`/`version`/`description`/`source`) and `id`; note search is not yet available.
- Example: `examples/data-sources/stackweaver_ansible_collections/data-source.tf` - list pre-installed
  collections and output their names.

## Divergences from upstream / TFE

Native data source - no TFE equivalent. Read-only/external: the collection set reflects the runner
image, not a managed resource. Galaxy `search` and per-job collections are intentionally out of scope
here (search is a backend placeholder; per-job returns the pre-installed set today).
