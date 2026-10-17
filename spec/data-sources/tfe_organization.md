<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_organization
tfe_alias: tfe_organization
kind: data-source
family: organizations
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: implemented
upstream_file: internal/provider/data_source_organization.go
go_tfe_type: Organization
compat_doc: docs/internal/tfe-compatibility/data-sources/organizations/tfe_organization.md
---
# stackweaver_organization (data source)

Reads one organization by name: its UUID (`external_id`), its default project and its policy
settings. Held out of v0.1 with the resource; the backing read is end-to-end proven in the monorepo.

## Client approach

`go-tfe-clean`. `Organizations.Read` → `GET /api/v2/organizations/:name`, stock shape. Registration
only: `"tfe_organization": dataSourceTFEOrganization()` in the SDKv2 `DataSourcesMap`.

## Schema

As upstream. `name` is Optional (falls back to the provider's `organization`); every other attribute
is Computed: `external_id`, `email`, `collaborator_auth_policy`, `cost_estimation_enabled`,
`default_project_id`, `owners_team_saml_role_id`, `two_factor_conformant`,
`send_passing_statuses_for_untriggered_speculative_plans`, `aggregated_commit_status_enabled`,
`assessments_enforced`, `speculative_plan_management_enabled`, `enforce_hyok`, `max_ttl_enabled`.

## Wire contract

- **Read:** `Organizations.Read` → `GET /api/v2/organizations/:name`. Readable with an
  organization-scoped token for its own organization, or a user token for any organization the user
  belongs to.

## Acceptance criteria (these ARE the test)

1. In the resource fixture, `data "stackweaver_organization"` on the created organization returns its
   `external_id` (a UUID), `email`, `default_project_id` equal to the resource's, and every policy
   flag the resource set.
2. The `tfe_organization` alias returns the same values.
3. With `name` omitted, it reads the provider's `organization`.

## Runtime criterion

CRUD-only (read-only data source).

## Docs + example

- `docs/data-sources/organization.md`; example `examples/data-sources/stackweaver_organization/data-source.tf`.

## Divergences from upstream / TFE

`two_factor_conformant` is always `false` (no identity-provider MFA signal yet);
`owners_team_saml_role_id` is always empty; `enforce_hyok` and `max_ttl_enabled` are always `false`.
