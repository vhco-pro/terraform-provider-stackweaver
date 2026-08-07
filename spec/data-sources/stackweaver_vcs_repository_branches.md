<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_vcs_repository_branches
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
# stackweaver_vcs_repository_branches

**Native data source - no TFE equivalent.** Lists the branches of one repository reachable through a
Stackweaver VCS connection. Read-only discovery helper: given `vcs_connection_id` + `owner` + `repo`, it
enumerates the repo's branches (name, head commit SHA, protection flag) so a config can pin a
`vcs_branch` without hardcoding it. Terraform-side counterpart to the branch dropdown in the UI's
`VcsRepoBranchPicker`. Branch shape: `core/services/vcs/provider.go` (`Branch`).

## Client approach

`native-client`. Not in `go-tfe`; served by the `internal/stackweaver` `VCS` service `Branches` (list)
method calling the Stackweaver VCS-connection API over HTTP. Read-only.

**Envelope:** plain JSON, not JSON:API. The handler returns `{ "data": [ <Branch>, ... ], "meta": {
"pagination": { "page", "per_page" } } }`, where each `Branch` is `{ "name": string, "commit": { "sha":
string, "url": string }, "protected": bool }`. The native client unmarshals this plain shape directly.
Confirm keys against `ListBranches` in `backend/internal/api/v2/handlers/vcs_connections.go` and the
`Branch` struct at implement time.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `vcs_connection_id` | string (uuid) | Required | - | - | no | the connection |
| `owner` | string | Required | - | - | no | org/user/project (left half of `full_name`) |
| `repo` | string | Required | - | - | no | repository name (right half of `full_name`) |
| `id` | string | Computed | - | - | no | synthetic id = `"<connection_id>/<owner>/<repo>"` |
| `branches` | list(object) | Computed | - | - | no | one object per branch |
| `branches[].name` | string | Computed | - | - | no | branch name |
| `branches[].commit_sha` | string | Computed | - | - | no | head commit SHA (`commit.sha`) |
| `branches[].protected` | bool | Computed | - | - | no | protection flag |

## Wire contract

- **Read (list):** `VCS.ListBranches(connID, owner, repo, opts)` →
  `GET /vcs-connections/:id/repositories/:owner/:repo/branches`. Pagination via `?page[...]`/`?per_page`
  (server default 30, cap 100). Response `data` is a plain array of `Branch`; accumulate all pages into
  `branches`, flattening nested `commit.sha` into `branches[].commit_sha`. Set synthetic `id` to
  `"<connection_id>/<owner>/<repo>"`.
- No create/update/delete - data source.
- **Provider-capability caveat:** providers without branch listing return **501 Not Implemented**; the
  native client surfaces it as an error, not an empty list.
- **Envelope:** plain JSON (`{ "data": [...], "meta": {...} }`). Native client owns marshalling.

## Acceptance criteria (these ARE the test)

1. `apply` of `{ vcs_connection_id, owner, repo }` for a known dev-stack repo lists branches and
   succeeds; `id` equals `"<connection_id>/<owner>/<repo>"`.
2. `branches` is a non-empty list; the repo's default branch (e.g. `main`) appears by name - assert
   `contains([for b in branches : b.name], "main")`.
3. Each `branches[]` object round-trips a non-empty `name` and a non-empty `commit_sha`.
4. Pointing at a non-existent `owner`/`repo` (or bad `vcs_connection_id`) fails the read with an error
   (not an empty result).
5. A provider that does not support branch listing surfaces the 501 as an error, not an empty
   `branches` list.

## Runtime criterion

`CRUD-only` (read-only). No runtime side effect. Correctness criterion: a `branches[].name` it returns
is directly usable as `vcs_branch` on `stackweaver_ansible_playbook` (or `ref` on
`stackweaver_vcs_yaml_files`).

## Docs + example

- Provider docs page: `docs/data-sources/vcs_repository_branches.md` - arguments (`vcs_connection_id`,
  `owner`, `repo`), computed `branches` list with nested attributes.
- Example: `examples/data-sources/stackweaver_vcs_repository_branches/data-source.tf` - feed
  `data.stackweaver_vcs_repositories.x.repositories[0].full_name` (split into owner/repo) and output the
  branch names.

## Divergences from upstream / TFE

Native data source - no TFE equivalent. Plain-JSON envelope. Nested `commit.sha` is flattened to a
top-level `commit_sha` in the Terraform schema for ergonomics; document the mapping.
