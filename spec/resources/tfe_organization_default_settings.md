<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_organization_default_settings
tfe_alias: tfe_organization_default_settings
kind: resource
family: organizations
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_organization_default_settings.go
go_tfe_type: Organization (OrganizationUpdateOptions)
compat_doc: docs/internal/tfe-compatibility/resources/organizations/tfe_organization_default_settings.md
---
# stackweaver_organization_default_settings

The organization-wide default workspace execution settings — default execution mode and default agent
pool — that sit at the top of the `workspace -> project -> organization` inheritance chain. A workspace
(or project) that expresses no preference of its own resolves these at run time. Maps onto Stackweaver's
`Organization.DefaultExecutionMode` / `DefaultAgentPoolID`.

## Client approach

`go-tfe-clean`. The resource is Plugin-Framework
(`internal/provider/resource_tfe_organization_default_settings.go`) and **has no object of its own**: it
is a view onto the organization. Create/Update/Delete all call `Organizations.Update`
(`:154,:195,:263`) and Read calls `Organizations.Read` (`:224`). Stackweaver added the two fields to the
existing `GET`/`PATCH /organizations/:name` and accepts the stock go-tfe wire shape unchanged
(`docs/internal/tfe-compatibility/resources/organizations/tfe_organization_default_settings.md`); no
wrapper and no new routes.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `organization` | string | Optional+Computed | yes | provider default | no | `RequiresReplace`; the URL segment `:name` |
| `default_execution_mode` | string | Required | yes | — | no | `RequiresReplace`; one of `remote`/`agent`/`local` |
| `default_agent_pool_id` | string | Optional | yes | — | no | `RequiresReplace`; required when mode is `agent`, forbidden otherwise |

## Wire contract

- **Create:** `Organizations.Update(org, OrganizationUpdateOptions{DefaultExecutionMode, DefaultAgentPool})`
  → `PATCH /organizations/:name`.
- **Read:** `Organizations.Read(org)` → `GET /organizations/:name`; reads back `DefaultExecutionMode`
  and, if present, `DefaultAgentPool.ID`.
- **Update:** never happens in place — every attribute is `RequiresReplace`, so a change destroys +
  recreates (which is still two `Organizations.Update` calls under the hood).
- **Delete:** `Organizations.Update(org, {DefaultExecutionMode: "remote", DefaultAgentPool: nil})` →
  resets the org to system defaults. There is **no** by-id object to remove.
- **JSON:API type:** `organizations`. **Asymmetric carriage (do not "tidy"):** on
  `OrganizationUpdateOptions` (`go-tfe/v1.go:8380,8383`) `default_execution_mode` is an **attribute**
  (`attr,default-execution-mode,omitempty`) but `default_agent_pool_id` is a **relationship**
  (`relation,default-agent-pool,omitempty`) — unlike `stackweaver_project`, which carries
  `default-agent-pool-id` as a plain attribute. An unset mode **must** read back as `remote`, never
  `""`, or the provider sees drift on the next plan.

## Acceptance criteria (these ARE the test)

1. `apply` of `{default_execution_mode = "agent", default_agent_pool_id = <pool>}` sets both on the org;
   on read `default_execution_mode` and `default_agent_pool_id` round-trip into state.
2. Re-`plan` after apply shows **no drift** (in particular an org whose mode is unset reads back
   `remote`, not `""`).
3. Because there is no object of its own, the E2E contract is **state-toggle**, not lifecycle: while
   applied, `GET /organizations/:name` reports the setting present; after `destroy` the org reports
   `default_execution_mode = remote` and no default agent pool (assert effect-present → effect-reverted,
   **not** a 404).
4. Changing `default_execution_mode` (or `organization`, or `default_agent_pool_id`) recreates — every
   attribute is `RequiresReplace`.
5. Setting `default_agent_pool_id` without `default_execution_mode = "agent"` fails validation
   (provider `ValidateConfig`: "Default execution mode must be set to 'agent' when
   default_agent_pool_id is set"); the server independently 422s the same invalid combination.
6. A pool that does not exist or belongs to another organization is rejected (422) — tenant safety.

## Runtime criterion

Real inheritance, not CRUD-only. `resolveEffectiveAgentPool`
(`backend/internal/api/v2/handlers/terraform/runs.go`) resolves execution settings down
`workspace -> project -> organization` at run creation, and this resource is its top level. Verified
live: with the org defaulting to `agent` + a pool and the project not overwriting, a real run persisted
`agent_pool_id` equal to the org's default pool (`inherited_from_org = t`). A workspace/project defers
only when its own mode is unset or `remote`; a project that overwrote a non-agent mode does not fall
through to the org agent default.

## Docs + example

- Provider docs page: `docs/resources/organization_default_settings.md` — arguments
  (`organization`, `default_execution_mode`, `default_agent_pool_id`), the agent-mode/pool constraint,
  the "view onto the organization / no object of its own" note, and import by organization name.
- Example: `examples/resources/stackweaver_organization_default_settings/resource.tf` — org defaulting
  to `agent` with an agent pool.

## Divergences from upstream / TFE

None on the attributes. TFE also exposes org-level assessment/session settings on the organization that
this resource does not cover — and neither do we, so there is no gap relative to the provider's
`tfe_organization_default_settings` surface. Drop-in.
