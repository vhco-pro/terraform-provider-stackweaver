<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_playbook
tfe_alias: n/a
kind: resource
family: ansible
origin: native
backing_api: implemented
client_approach: native-client
status: spec'd
upstream_file: n/a (native - no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + core/models/ansible_playbook.go)
---
# stackweaver_ansible_playbook

**Native resource - no TFE equivalent.** Registers an Ansible playbook: a named pointer, within a
project, at a playbook file in a VCS repository. Job templates and jobs reference it; a sync action
pulls the repo. Model: `core/models/ansible_playbook.go` (`AnsiblePlaybook`).

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `AnsiblePlaybooks` service
(List/Create/Read/Update/Delete + a `Sync` action) calling the Stackweaver Ansible API over HTTP.
The API is **JSON:API-shaped** (`data.attributes`, per `handlers/ansible/playbooks.go`) - as most of
the native Ansible surface is. NOTE: the native surface is **mixed** (most resources are JSON:API with
snake- or dash-cased keys; a few list/discovery endpoints are plain JSON), so the native client must
not assume one envelope - see `plan/plan.md` and each spec's wire contract.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string (uuid) | Computed | - | - | no | server-assigned |
| `project_id` | string (uuid) | Required | yes | - | no | owning project; `(project_id,name)` unique |
| `name` | string | Required | no | - | no | unique within the project |
| `description` | string | Optional | no | `""` | no | |
| `vcs_connection_id` | string (uuid) | Optional | no | - | no | the VCS connection to pull from |
| `vcs_repository` | string | Optional | no | - | no | `"owner/repo"` |
| `vcs_branch` | string | Optional+Computed | no | `main` | no | |
| `playbook_path` | string | Optional+Computed | no | `site.yml` | no | path within the repo |
| `source_mode` | string | Optional+Computed | no | `cached` | no | `cached` \| `fresh` |
| `last_sync_at` / `last_sync_status` / `last_sync_commit` / `cached_commit` / `cached_at` | (various) | Computed | - | - | no | sync-status readback |

## Wire contract

- **Create:** `POST /organizations/:org/ansible/playbooks` - body: `project_id`, `name`,
  `description?`, `vcs_connection_id?`, `vcs_repository?`, `vcs_branch?`, `playbook_path?`,
  `source_mode?`.
- **Read:** `GET /ansible/playbooks/:id`.
- **Update:** `PATCH /ansible/playbooks/:id` (name/description/vcs/path/source_mode in place).
- **Delete:** `DELETE /ansible/playbooks/:id`.
- **Action (not part of CRUD):** `POST /ansible/playbooks/:id/actions/sync` - pulls the repo. Exposed
  as an optional resource action / not a required lifecycle step; document, do not force on apply.
- **Envelope:** JSON:API-shaped (`data.attributes`). Native client owns marshalling.

## Acceptance criteria (these ARE the test)

1. `apply` of `{project_id, name, vcs_connection_id, vcs_repository, playbook_path}` creates the
   playbook; `id`, `name`, `playbook_path`, `source_mode` round-trip into state.
2. Re-`plan` after apply shows **no drift** (computed sync-status fields must not cause perpetual diff).
3. `playbook_path`/`vcs_branch`/`source_mode` defaults settle to `site.yml`/`main`/`cached` when omitted.
4. Updating `description`/`playbook_path` applies in place; changing `project_id` forces recreate.
5. `destroy` removes it; a subsequent `GET /ansible/playbooks/:id` returns 404.

## Runtime criterion

The playbook is a declarative source pointer - its runtime effect is that job templates / jobs
resolve it and the `sync` action fetches the repo at the pinned branch/path. Verified: create a
playbook + launch (or dry-run) a job template referencing it, and the referenced source resolves.
Config-with-real-effect, not dead CRUD.

## Docs + example

- Provider docs page: `docs/resources/ansible_playbook.md`.
- Example: `examples/resources/stackweaver_ansible_playbook/resource.tf` - a project + a VCS-backed
  playbook (referencing a `tfe_github_app_installation` / VCS connection id).

## Divergences from upstream / TFE

Native resource - no TFE equivalent. `sync` is a Stackweaver action with no Terraform analogue
(surfaced as an optional action, not a lifecycle requirement).
