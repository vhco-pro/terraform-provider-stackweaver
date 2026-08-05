<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_vcs_playbook_files
tfe_alias: n/a
kind: data-source
family: ansible
origin: native
backing_api: implemented
client_approach: native-client
status: spec'd
upstream_file: n/a (native — no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + backend/internal/api/v2/handlers/ansible/playbook_discovery.go)
---
# stackweaver_ansible_vcs_playbook_files

**Native data source — no TFE equivalent.** A read-only discovery helper that lists the playbook
candidate files in a connected VCS repository at a branch, each annotated with whether it is already
registered as a `stackweaver_ansible_playbook`. This is the same listing the bulk-import wizard and
the job-template repository browser consume; use it to drive a `for_each` over discovered playbooks.
Handler: `backend/internal/api/v2/handlers/ansible/playbook_discovery.go` (`ListPlaybookFiles`).

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `AnsiblePlaybooks` service
(or a `Discovery` sub-service) with a `ListVCSPlaybookFiles(ctx, org, opts)` read method calling
`GET /organizations/:org/ansible/vcs-playbook-files` over HTTP. The endpoint returns a plain-JSON
envelope (`{"data": [ ... ]}`), **not** JSON:API — the native client marshals accordingly. This is a
discovery/observability helper, read-only: no create/update/delete.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `organization` | string | Optional+Computed | — | provider default | no | org name; falls back to the provider default |
| `vcs_connection_id` | string (uuid) | Required | — | — | no | the VCS connection to browse; must belong to the org |
| `repository` | string | Required | — | — | no | `"owner/repo"`; rejected if it lacks a `/` |
| `branch` | string | Required | — | — | no | required — the listing and annotation are branch-scoped; there is no silent default |
| `path` | string | Optional | — | `""` | no | scope prefix; when set, only files under this repo path are returned |
| `id` | string | Computed | — | — | no | synthesized (e.g. `org/connection_id/repository/branch`); no server-assigned id |
| `files` | list(object) | Computed | — | — | no | discovered candidates; `{path, name, registered, playbook_id, playbook_name}` |
| `files[].path` | string | Computed | — | — | no | repo-relative file path |
| `files[].name` | string | Computed | — | — | no | base filename |
| `files[].registered` | bool | Computed | — | — | no | whether a playbook is already registered for this path |
| `files[].playbook_id` | string (uuid) | Computed | — | — | no | present only when `registered` is true |
| `files[].playbook_name` | string | Computed | — | — | no | present only when `registered` is true |

## Wire contract

- **Read/lookup:** `AnsiblePlaybooks.ListVCSPlaybookFiles(ctx, org, {vcs_connection_id, repository,
  branch, path?})` → `GET /organizations/:org/ansible/vcs-playbook-files?vcs_connection_id=&repository=&branch=&path=`.
- **Create/Update/Delete:** n/a — read-only data source.
- **Envelope:** plain JSON, `{"data": [ {"path", "name", "registered", "playbook_id"?,
  "playbook_name"?} ]}`. Native client owns marshalling. Filtering excludes conventional
  non-playbook directories (`roles/`, `group_vars/`, …) and well-known non-playbook YAML
  (`requirements.yml`, `galaxy.yml`, …); results are sorted for stable output.
- **Required-input errors:** missing/invalid `repository` (not `owner/repo`) → 400; missing `branch`
  → 400; connection not in the org → 404. A provider whose VCS backend cannot list files returns 501
  (`"not implemented"`).

## Acceptance criteria (these ARE the test)

Assert against known dev-stack state: a VCS connection + a repository known to contain at least one
playbook (e.g. the fixture's `site.yml`), on a known branch.

1. Reading `data.stackweaver_ansible_vcs_playbook_files` with `{vcs_connection_id, repository, branch}`
   for the fixture repo returns a non-empty `files` list; the known playbook path (e.g. `site.yml`)
   appears with matching `path`/`name`.
2. The computed `id` is set (synthesized from the inputs).
3. For a path already registered via `stackweaver_ansible_playbook`, that entry's `registered` is
   `true` and `playbook_id`/`playbook_name` match the registered playbook; unregistered candidates
   have `registered = false`.
4. Setting `path` to a subdirectory scopes `files` to entries under that prefix.
5. Re-`plan` after apply shows **no drift** (`files`, `id` are Computed-only).
6. Omitting `branch` (or passing a `repository` without a `/`) surfaces the backend 400 as a plan/apply
   error.

## Runtime criterion

Read-only discovery helper. Resolves the repository file listing at the pinned branch into a `files`
collection annotated with registration state; no runtime side effect beyond the VCS list read. Its
real-world effect is enabling a `for_each` over discovered playbooks to drive
`stackweaver_ansible_playbook` registration.

## Docs + example

- Provider docs page: `docs/data-sources/ansible_vcs_playbook_files.md` — arguments `organization`,
  `vcs_connection_id`, `repository`, `branch`, `path`; computed `files` (each
  `path`/`name`/`registered`/`playbook_id`/`playbook_name`) and `id`.
- Example: `examples/data-sources/stackweaver_ansible_vcs_playbook_files/data-source.tf` — discover
  playbooks in a VCS-connected repo and `for_each` them into `stackweaver_ansible_playbook`.

## Divergences from upstream / TFE

Native data source — no TFE equivalent. Plain-JSON envelope (not JSON:API). Read-only discovery
surface with no lifecycle. `branch` is deliberately required (no silent default) so the listing,
annotation, and any subsequent import all refer to the same branch.
