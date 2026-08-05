<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_config
tfe_alias: n/a
kind: resource
family: ansible
origin: native
backing_api: implemented
client_approach: native-client
status: spec'd
upstream_file: n/a (native — no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + core/models/ansible_config.go)
---
# stackweaver_ansible_config

**Native resource — no TFE equivalent.** Manages the `ansible.cfg` content for a scope. There is **one
config per scope entity** (singleton): one row keyed by exactly one of organization / project /
workspace. Runs at that scope (and more-specific scopes that do not override it) use this content;
resolution priority is Workspace > Project > Organization. Model: `core/models/ansible_config.go`
(`AnsibleConfig`, ~line 14).

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `AnsibleConfig` service
(Get/Upsert/Delete per scope) calling the Stackweaver Ansible-config API over HTTP.

**Envelope is flat plain JSON** (verified in `handlers/ansible_config.go`): the request body is
`{"config_content":"..."}` (single required field, `binding:"required"`) and the response is
`{"data":{"id","type","scope","organization_id","project_id","workspace_id","config_content",
"created_at","updated_at"}}`. This is neither the JSON:API envelope of the inventory resources nor the
model-tag JSON of the playbook — it is a bespoke flat shape. The native client owns marshalling.

**Singleton-per-scope semantics.** The scope entity is chosen by the endpoint, not the body: PUT to the
org endpoint upserts the single org-scoped row, PUT to the project endpoint upserts the single
project-scoped row. There is no separate create — `PUT` is create-or-update. In Terraform this maps to
one resource instance per scope entity: create = first PUT, update = subsequent PUT, and the scope
selector is ForceNew (changing scope targets a different singleton).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string (uuid) | Computed | — | — | no | server-assigned |
| `organization` | string | Optional | **yes** | — | no | org **name**; set for org-scoped config; URL segment; maps to model `organization_id` |
| `project_id` | string (uuid) | Optional | **yes** | — | no | set for project-scoped config; URL segment |
| `config_content` | string | Required | no | — | no | not-null; the raw `ansible.cfg`; updatable in place |
| `scope` | string | Computed | — | — | no | server-returned: `organization` \| `project` \| `workspace` |
| `workspace_id` | string | Computed | — | — | no | echoed in response; **no REST route to set it** (see divergences) |
| `created_at` / `updated_at` | string (RFC3339) | Computed | — | — | no | |

Exactly one of `organization` / `project_id` must be set (validated by the provider — the API infers
scope from which endpoint is called).

## Wire contract

- **Create/Update (upsert):**
  - org scope: `PUT /organizations/:name/ansible-config` — body `{"config_content"}`.
  - project scope: `PUT /projects/:id/ansible-config` — body `{"config_content"}`.
  - Same endpoint handles both first-write and subsequent updates (no distinct POST).
- **Read:**
  - org scope: `GET /organizations/:name/ansible-config`.
  - project scope: `GET /projects/:id/ansible-config`.
  - `GET .../ansible-config/effective` resolves the merged/most-specific config; **read-only helper**,
    not this resource's state (data-source territory).
- **Delete:**
  - org scope: `DELETE /organizations/:name/ansible-config`.
  - project scope: `DELETE /projects/:id/ansible-config`.
- **Envelope:** flat plain JSON — request `{"config_content"}`, response
  `{"data":{"id","type","scope","organization_id","project_id","workspace_id","config_content",
  "created_at","updated_at"}}`. Native client owns marshalling.

## Acceptance criteria (these ARE the test)

1. `apply` of `{organization, config_content}` upserts the org-scoped config; `id`, `scope`
   (`organization`), `organization`, `config_content` round-trip into state.
2. `apply` of `{project_id, config_content}` upserts the project-scoped config; `scope` reads back
   `project`.
3. Re-`plan` after apply shows **no drift** — computed `id`/`scope`/`workspace_id`/`created_at`/
   `updated_at` must not cause a perpetual diff.
4. Updating `config_content` applies **in place** via the same PUT (verify the row's `id` is stable —
   upsert must not recreate).
5. Changing the scope selector (`organization` ↔ `project_id`) forces **recreate** (ForceNew) — it
   targets a different singleton.
6. Singleton enforcement: a second resource instance targeting the **same** scope entity converges to
   the same underlying row (last-write-wins on `config_content`), never a duplicate.
7. `destroy` removes it; a subsequent `GET .../ansible-config` returns 404.

## Runtime criterion

Config-with-real-effect. At run time the runner renders `ansible.cfg` from the most-specific config for
the run's scope (Workspace > Project > Org), so this content changes ansible behavior (e.g. host key
checking, callback plugins, forks). Verified: set an org config with an observable directive, launch
(or dry-run) a job whose scope resolves to it, and confirm the rendered `ansible.cfg` carries the
directive; `GET .../effective` returns this config when no more-specific one exists.

## Docs + example

- Provider docs page: `docs/resources/ansible_config.md` — document the singleton-per-scope model and
  the Workspace > Project > Org resolution order.
- Example: `examples/resources/stackweaver_ansible_config/resource.tf` — an org-scoped `ansible.cfg`
  and a project-scoped override using a heredoc `config_content`.

## Divergences from upstream / TFE

Native resource — no TFE equivalent. **Workspace-scope is not manageable by this resource:** the model
has `workspace_id` and the response echoes it, but the REST routes expose **only** org and project
scopes (`/organizations/:name/ansible-config`, `/projects/:id/ansible-config`) — there is no workspace
PUT/GET/DELETE route, so `workspace_id` is Computed-only and the resource covers org + project scopes.
`GET .../effective` is a read-only resolution helper (surface as a data source, not part of this
resource). Upsert-on-PUT (no separate create) is the create/update mechanism.
