<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_organizations
tfe_alias: tfe_organizations
kind: data-source
family: organizations
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: implemented
upstream_file: internal/provider/data_source_organizations.go
go_tfe_type: Organization
compat_doc: docs/internal/tfe-compatibility/data-sources/organizations/tfe_organizations.md
---
# stackweaver_organizations (data source)

Lists the organizations the token can see: `names`, and `ids` mapping each name to its UUID.

## Client approach

`go-tfe-clean`. `Organizations.List` → `GET /api/v2/organizations` (JSON:API, paginated), stock
shape. Registration only: `"tfe_organizations": dataSourceTFEOrganizations()` in the SDKv2
`DataSourcesMap`.

## Schema

As upstream: `names` (list of string, Computed), `ids` (map of string, Computed), `admin` (bool,
Optional, default false).

## Wire contract

- **Read:** `Organizations.List` → `GET /api/v2/organizations`. A user token lists every organization
  the user belongs to; an organization-scoped token lists only its own organization.
- `admin = true` calls TFE's site-admin listing (`GET /api/v2/admin/organizations`), which Stackweaver
  does not have.

## Acceptance criteria (these ARE the test)

1. In the resource fixture (user token), `names` contains the created organization and `ids[<name>]`
   equals the organization data source's `external_id`.
2. With an organization-scoped token, `names` is exactly that one organization.
3. The `tfe_organizations` alias returns the same values.

## Runtime criterion

CRUD-only (read-only data source).

## Docs + example

- `docs/data-sources/organizations.md` (states that `admin` is unsupported); example
  `examples/data-sources/stackweaver_organizations/data-source.tf`.

## Divergences from upstream / TFE

`admin = true` is unsupported: there is no site-admin organization listing. The provider docs say so;
the non-admin listing already returns every organization the caller can see.
