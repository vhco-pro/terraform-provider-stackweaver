<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_vcs_repositories
tfe_alias: n/a
kind: data-source
family: vcs
origin: native
backing_api: implemented
client_approach: native-client
status: spec'd
upstream_file: n/a (native - no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + core/services/vcs/provider.go)
---
# stackweaver_vcs_repositories

**Native data source - no TFE equivalent.** Lists the repositories reachable through a Stackweaver VCS
connection (GitHub App / OAuth / Azure DevOps). Read-only discovery helper: given a `vcs_connection_id`,
it enumerates the repositories that connection can see, so a config can pick a repo (its `full_name`,
default branch, clone URL) to wire into an `stackweaver_ansible_playbook` or a workspace VCS block. This
is the Terraform-side counterpart to the repo picker in the UI (`VcsRepoBranchPicker`). Repository shape:
`core/services/vcs/provider.go` (`Repository`).

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `VCS` service `Repositories`
(list) method calling the Stackweaver VCS-connection API over HTTP. Read-only.

**Envelope:** JSON:API. The handler answers a collection of `vcs-repositories` resources whose `id` is
the provider's numeric repository id as a string and whose attributes are `name`, `full-name`,
`description`, `private`, `default-branch`, `url`, `clone-url` and `ssh-url`. `meta.pagination` is the provider page block: `total-pages` and `total-count` are null (the provider
reports no total) and `next-page` is set only when the page came back full, so the native client
follows `next-page` until it is null. Confirm
against `ListRepositories` in `backend/internal/api/v2/handlers/vcs_connections.go` and
`VCSRepositoryAttributes`.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `vcs_connection_id` | string (uuid) | Required | - | - | no | the connection to enumerate |
| `project` | string | Optional | - | - | no | Azure DevOps project scope → `?project=`; ignored by providers without a project layer |
| `id` | string | Computed | - | - | no | synthetic id = `vcs_connection_id` |
| `repositories` | list(object) | Computed | - | - | no | one object per reachable repo |
| `repositories[].id` | number | Computed | - | - | no | provider numeric repo id |
| `repositories[].name` | string | Computed | - | - | no | short name |
| `repositories[].full_name` | string | Computed | - | - | no | `"owner/repo"` - the value fed to the branches/files data sources |
| `repositories[].description` | string | Computed | - | - | no | |
| `repositories[].private` | bool | Computed | - | - | no | |
| `repositories[].default_branch` | string | Computed | - | - | no | |
| `repositories[].url` / `repositories[].clone_url` / `repositories[].ssh_url` | string | Computed | - | - | no | |

## Wire contract

- **Read (list):** `VCS.ListRepositories(connID, opts)` → `GET /vcs-connections/:id/repositories`.
  Optional `?project=` for Azure DevOps scoping; pagination via `?page[number]`/`?page[size]` (server
  default 30, cap 100; the client asks for 100). Accumulate every page, following `next-page`, into
  `repositories`, parsing each resource `id` back to the numeric repository id. Set synthetic `id` to
  `vcs_connection_id`.
- No create/update/delete - data source.
- **Provider-capability caveat:** a provider that cannot list repositories returns **501 Not
  Implemented**; an unmaterialized Azure DevOps identity returns **403**. The native client must surface
  these as clear errors, not swallow them into an empty list.
- **Envelope:** JSON:API collection of `vcs-repositories` with the provider page block. Native client
  owns marshalling.

## Acceptance criteria (these ARE the test)

1. `apply` of `{ vcs_connection_id = <dev connection id> }` lists repositories and succeeds; `id` is set
   to the connection id.
2. `repositories` is a non-empty list against a dev connection with at least one reachable repo; each
   object round-trips `full_name` (matching `owner/repo`) and `default_branch` (non-empty).
3. A known dev-stack repository (e.g. the fixture's test repo) appears in `repositories` by its
   `full_name` - assert `contains([for r in repositories : r.full_name], "<owner/repo>")`.
4. Pointing at a non-existent `vcs_connection_id` fails the read with a 404-derived error (not an empty
   result).
5. A provider/connection that does not support repository listing surfaces the 501 as an error, not an
   empty `repositories` list.

## Runtime criterion

`CRUD-only` (read-only). No runtime side effect. Its correctness criterion is that a `full_name` it
returns is directly usable as the `owner`/`repo` inputs of `stackweaver_vcs_repository_branches` /
`stackweaver_vcs_yaml_files` (the discovery chain resolves end to end).

## Docs + example

- Provider docs page: `docs/data-sources/vcs_repositories.md` - argument (`vcs_connection_id`,
  `project`), computed `repositories` list with nested attributes.
- Example: `examples/data-sources/stackweaver_vcs_repositories/data-source.tf` - list repos for a
  connection and output the set of `full_name`s.

## Divergences from upstream / TFE

Native data source - no TFE equivalent (TFE has no VCS-connection repository-listing surface;
`tfe_oauth_client` never enumerates repos). `project` argument only meaningful for
Azure DevOps.
