<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_vcs_yaml_files
tfe_alias: n/a
kind: data-source
family: vcs
origin: native
backing_api: implemented
client_approach: native-client
status: spec'd
upstream_file: n/a (native — no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + backend/internal/api/v2/handlers/vcs_connections.go)
---
# stackweaver_vcs_yaml_files

**Native data source — no TFE equivalent.** Lists candidate YAML files (or, in inventory mode, inventory
files) inside a repository reachable through a Stackweaver VCS connection, at an optional branch/ref.
Read-only discovery helper for picking a `playbook_path` (or an inventory path) to wire into a
`stackweaver_ansible_playbook`: given `vcs_connection_id` + `owner` + `repo`, it returns the matching
file paths so a config does not have to hardcode `site.yml`. Terraform-side counterpart to the file
picker in the UI. Returned value is a flat list of repo-relative path strings.

## Client approach

`native-client`. Not in `go-tfe`; served by the `internal/stackweaver` `VCS` service `Files` (list)
method calling the Stackweaver VCS-connection API over HTTP. Read-only.

**Envelope:** plain JSON, not JSON:API — and notably **not** paginated. Both endpoints return
`{ "data": [ "path/one.yml", "path/two.yaml", ... ] }` — `data` is a flat array of path strings, no
`meta`. The native client unmarshals a `[]string`. Confirm against `ListYamlFiles` /
`ListInventoryFiles` in `backend/internal/api/v2/handlers/vcs_connections.go` (both delegate to
`provider.ListFiles(..., extensions)`).

## Endpoint note (route correction)

The two backing routes are nested **under the repository**, not directly under the connection:

- `GET /vcs-connections/:id/repositories/:owner/:repo/yaml-files` — extensions `.yaml`, `.yml`
- `GET /vcs-connections/:id/repositories/:owner/:repo/inventory-files` — extensions `.ini`, `.yaml`,
  `.yml`, `.json`

Both require `owner` + `repo` path params (the plan's `GET /vcs-connections/:id/yaml-files` shorthand is
not the real path). Both accept an optional `?ref=<branch>` query param. This single data source covers
**both** via a `file_type` argument — `"playbook"` (default → `yaml-files`) or `"inventory"` (→
`inventory-files`) — rather than a separate sibling data source, because the two differ only in the
server-side extension filter and share identical inputs/output shape. Documented here as the deliberate
choice.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `vcs_connection_id` | string (uuid) | Required | — | — | no | the connection |
| `owner` | string | Required | — | — | no | org/user/project |
| `repo` | string | Required | — | — | no | repository name |
| `ref` | string | Optional | — | provider/repo default branch | no | branch or commit → `?ref=`; empty = default branch |
| `file_type` | string | Optional | — | `playbook` | no | `playbook` → `yaml-files`; `inventory` → `inventory-files` |
| `id` | string | Computed | — | — | no | synthetic id = `"<connection_id>/<owner>/<repo>/<file_type>@<ref>"` |
| `paths` | list(string) | Computed | — | — | no | repo-relative matching file paths |

## Wire contract

- **Read (list, playbook mode):** `VCS.ListYamlFiles(connID, owner, repo, ref)` →
  `GET /vcs-connections/:id/repositories/:owner/:repo/yaml-files?ref=<ref>`. `data` is `[]string` →
  `paths`.
- **Read (list, inventory mode):** `VCS.ListInventoryFiles(connID, owner, repo, ref)` →
  `GET /vcs-connections/:id/repositories/:owner/:repo/inventory-files?ref=<ref>`. `data` is `[]string`
  → `paths`.
- Not paginated — a single response holds the full list. Set synthetic `id` to
  `"<connection_id>/<owner>/<repo>/<file_type>@<ref>"`.
- No create/update/delete — data source.
- **Provider-capability caveat:** providers without file listing return **501 Not Implemented**; the
  native client surfaces it as an error, not an empty list.
- **Envelope:** plain JSON `{ "data": [ "<path>", ... ] }`, no `meta`. Native client owns marshalling.

## Acceptance criteria (these ARE the test)

1. `apply` of `{ vcs_connection_id, owner, repo }` (default `file_type = "playbook"`) against a known
   dev-stack repo containing at least one playbook lists `.yml`/`.yaml` paths and succeeds; `id`
   reflects connection/owner/repo/file_type/ref.
2. `paths` is a non-empty list for a repo with playbooks; a known playbook (e.g. `site.yml`) appears —
   assert `contains(paths, "site.yml")` (or the fixture repo's actual playbook path).
3. Every returned path ends in `.yml`/`.yaml` in `playbook` mode; in `inventory` mode, paths end in one
   of `.ini`/`.yaml`/`.yml`/`.json`.
4. Switching `file_type` to `"inventory"` targets the `inventory-files` endpoint and returns the
   inventory-extension set (asserts the two modes hit different routes).
5. Supplying a `ref` for a branch known to contain a differently-named playbook returns that branch's
   paths (asserts `?ref=` is honored); an empty `ref` reads the default branch.
6. A non-existent `owner`/`repo` (or a provider without file listing) fails with an error, not an empty
   `paths` list.

## Runtime criterion

`CRUD-only` (read-only). No runtime side effect. Correctness criterion: a path it returns is directly
usable as `playbook_path` on `stackweaver_ansible_playbook` (playbook mode) or as an inventory source
(inventory mode) — the pick-a-playbook discovery chain resolves end to end.

## Docs + example

- Provider docs page: `docs/data-sources/vcs_yaml_files.md` — arguments (`vcs_connection_id`, `owner`,
  `repo`, `ref`, `file_type`), computed `paths`; explain `file_type` toggles yaml-files vs
  inventory-files.
- Example: `examples/data-sources/stackweaver_vcs_yaml_files/data-source.tf` — list playbooks in a repo
  at a branch and feed `paths[0]` into a `stackweaver_ansible_playbook`.

## Divergences from upstream / TFE

Native data source — no TFE equivalent. Plain-JSON, unpaginated `[]string` envelope. The
`inventory-files` variant is folded into this data source via `file_type` rather than shipped as a
separate `stackweaver_vcs_inventory_files` data source (documented choice; revisit if the two endpoints'
inputs ever diverge). Backing routes are nested under `/repositories/:owner/:repo/`, correcting the
plan's `/vcs-connections/:id/yaml-files` shorthand.
