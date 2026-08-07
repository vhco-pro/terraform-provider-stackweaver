<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_terraform_version
tfe_alias: tfe_terraform_version
kind: resource
family: versions
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_terraform_version.go
go_tfe_type: AdminTerraformVersion
compat_doc: docs/internal/tfe-compatibility/resources/versions/tfe_terraform_version.md
---
# stackweaver_terraform_version

Registers a Terraform version (binary URL + checksum + availability flags) in the platform's admin-level
version catalog. Site administrators use it to control which versions workspaces may run. Maps onto
Stackweaver's `TerraformVersion` model, served by the admin API.

## Client approach

`go-tfe-clean`. The resource is Plugin-Framework
(`internal/provider/resource_tfe_terraform_version.go`) driving `go-tfe`'s
`Admin.TerraformVersions.Create/Read/Update/Delete` verbatim. Stackweaver serves the stock
`terraform-versions` JSON:API shape unchanged
(`docs/internal/tfe-compatibility/resources/versions/tfe_terraform_version.md`); no wrapper. Access is
restricted to an "owners" team (site admin); non-admins receive **404** (matching TFE's practice of
hiding admin endpoints).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | - | - | no | `tool-`-prefixed primary id; `UseStateForUnknown` |
| `version` | string | Required | no | - | no | semver, e.g. `1.13.0`; updatable in place |
| `url` | string | Optional+Computed | no | - | no | linux/amd64 binary URL; empty → null-normalized; synced with amd64 arch |
| `sha` | string | Optional+Computed | no | - | no | SHA-256; empty → null-normalized; synced with amd64 arch |
| `official` | bool | Optional+Computed | no | `false` | no | set for auto-seeded versions |
| `enabled` | bool | Optional+Computed | no | `true` | no | availability for workspaces |
| `beta` | bool | Optional+Computed | no | `false` | no | |
| `deprecated` | bool | Optional+Computed | no | `false` | no | |
| `deprecated_reason` | string | Optional | no | - | no | see null-handling note below |
| `archs` | set(obj{url,sha,os,arch}) | Optional+Computed | no | - | no | per-arch download rows; `UseStateForUnknown` + preserve-amd64 |

## Wire contract

- **Create:** `Admin.TerraformVersions.Create(AdminTerraformVersionCreateOptions)` →
  `POST /admin/terraform-versions`. Attrs: `version` (required), `url?`, `sha?`, `official?`,
  `enabled?`, `beta?`, `deprecated?`, `deprecated-reason?`, `archs?`.
- **Read:** `Admin.TerraformVersions.Read(id)` → `GET /admin/terraform-versions/:id`.
- **Update:** `Admin.TerraformVersions.Update(id, AdminTerraformVersionUpdateOptions)` →
  `PATCH /admin/terraform-versions/:id`. All fields update in place - **nothing is ForceNew**, including
  `version`.
- **Delete:** `Admin.TerraformVersions.Delete(id)` → `DELETE /admin/terraform-versions/:id`. Refused for
  `official` versions and versions in use by a workspace (matches TFE).
- **JSON:API type:** `terraform-versions`; id prefix `tool-`. **`deprecated-reason` null-handling
  (matches go-tfe `*string,omitempty`):** the field is `DeprecatedReason *string
  jsonapi:"attr,deprecated-reason,omitempty"` on `AdminTerraformVersion`/CreateOptions/UpdateOptions
  (`go-tfe/v1.go:1996,2033,2048`). The provider sends `tfe.String(...)` even when the config leaves it
  unset (so `""` goes on the wire), and on read maps `nil → types.StringNull()` (`:216-220,:265-269`).
  Stackweaver therefore **treats an empty `deprecated-reason` as nil and omits it from the response**,
  so the framework reads back null and does not raise a "provider produced inconsistent result" error.
  `url` and `sha` follow the same empty-string → null normalization (`:221-230,:270-279`).

## Acceptance criteria (these ARE the test)

1. `apply` of `{version, url, sha, official=false, enabled=true, beta=false}` creates the version; `id`
   (with `tool-` prefix), `version`, `url`, `sha`, `official`, `enabled`, `beta`, `deprecated` all
   round-trip into state.
2. Re-`plan` after apply shows **no drift** - specifically, with `deprecated_reason` unset the read
   returns null (not `""`) and does not produce an inconsistent-result error; likewise empty `url`/`sha`
   read back as null.
3. Updating `enabled` (or `deprecated` + `deprecated_reason`, or `version`) applies **in place** - no
   attribute is ForceNew, so no recreate.
4. Setting `deprecated = true, deprecated_reason = "superseded by 1.14"` round-trips the reason; then
   clearing `deprecated_reason` reads back null with no drift.
5. `destroy` removes it; a subsequent `Admin.TerraformVersions.Read(id)` returns not-found.
6. Delete of an `official` version, and delete of a version currently in use by a workspace, are both
   refused; a non-admin caller receives 404 on every endpoint.

## Runtime criterion

Not CRUD-only: the catalog gates real execution. A workspace's version resolves as
workspace `terraform-version` → organization `default-terraform-version` → error if neither is set;
there is no hardcoded fallback. At run time the runner uses the exact resolved version, downloading the
binary from the registered `url` (or `releases.hashicorp.com`) if it is not present locally. Verified:
after registering a custom version and pointing a workspace at it, a run executes with that exact
binary; a disabled version is not selectable.

## Docs + example

- Provider docs page: `docs/resources/terraform_version.md` - arguments (`version`, `url`, `sha`,
  `official`, `enabled`, `beta`, `deprecated`, `deprecated_reason`, `archs`), computed `id`, admin-only
  access note, delete constraints (official / in-use protected), import by id or by version string.
- Example: `examples/resources/stackweaver_terraform_version/resource.tf` - a custom non-official
  version with url + sha, enabled.

## Divergences from upstream / TFE

None. Drop-in with `tfe_terraform_version`. The `deprecated-reason`/`url`/`sha` empty-string → null
normalization is a compatibility *fix* that keeps the go-tfe `*string,omitempty` contract intact, not a
divergence. Stackweaver auto-seeds official versions (1.5.x–1.13.x) on startup; those appear as
`official = true` resources and are delete-protected, consistent with TFE.
