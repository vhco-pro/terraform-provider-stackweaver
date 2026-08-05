<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
# Implementation plan — terraform-provider-stackweaver

The **HOW**, decided and reviewed once, before the fan-out. Pairs with the **WHAT** in `spec/`
(per-resource schema, wire contract, acceptance criteria) and the ordered worklist in
`plan/tasks.md`. Spec-driven order: **spec → plan → tasks → implement**; nothing builds until this
plan is approved.

## Key realization (reframes the effort)

The forked resources are **already implemented** — the resource logic is upstream
`terraform-provider-tfe` code, forked at v0.79.0 and driving stock `go-tfe`, which the spec verdict
confirmed works unchanged against Stackweaver's API. So for the 39 v0.1 resources, "implement" is
**not codegen** — it is: register the resource under the `stackweaver_*` + `tfe_*` names, generate an
acceptance fixture from the spec's criteria, run it against the dev stack, write docs, open a PR. The
only genuinely new Go code in v0.1 is **provider-level** (done once): the provider rename, the client
host default, the alias layer, and the strip of unsupported resources. Heavy codegen is deferred to
the **native** surface (Phase 5), which uses `internal/stackweaver`.

This is exactly why the plan phase matters: it turns "build 61 resources" into "one provider-plumbing
task + 61 register-verify-document tasks", which is far smaller and lower-risk than it first looked.

## 1. Provider identity + client host

- **Address:** `main.go:22` `tfeProviderName = "registry.terraform.io/hashicorp/tfe"` →
  `"registry.terraform.io/vhco-pro/stackweaver"`. Primary provider type name becomes `stackweaver`.
- **Client host:** the provider configures a `go-tfe` client from the `hostname`/`token` provider
  args (and `TFE_HOSTNAME`/`TFE_TOKEN`). No client change — it already targets an arbitrary host;
  we only update defaults/docs/UA string to Stackweaver. `go-tfe` stays an unmodified dependency.

## 2. Alias mechanism (`stackweaver_*` primary + `tfe_*` alias)

The provider is **muxed**: SDKv2 (`provider.go`, `ResourcesMap`/`DataSourcesMap`, string keys) +
plugin-framework (`provider_next.go`, `Resources()`/`DataSources()` factory slices; each resource's
`Metadata` sets `resp.TypeName = req.ProviderTypeName + "_<name>"`). Both must expose both prefixes.

- **SDKv2:** in the maps, register each supported resource under **both** keys —
  `"stackweaver_<name>"` and `"tfe_<name>"` → the same factory. Trivial, one line each.
- **Framework:** set the framework provider `Metadata` `TypeName = "stackweaver"` (so real resources
  are `stackweaver_*`). For each supported framework resource, also register a thin **alias wrapper**
  that embeds the real `resource.Resource` and overrides **only** `Metadata` to emit
  `"tfe_<name>"` — every other method (Schema/Create/Read/Update/Delete/ImportState) delegates to the
  embedded resource. One tiny generic wrapper type (`aliasResource{inner, typeName}`) covers all of
  them; same pattern for `aliasDataSource`.
- This is the **one intentional seam** vs. upstream: a single `alias.go` file plus the two edited
  registration lists. Individual `resource_tfe_*.go` files stay byte-identical to upstream, so
  backports to them merge cleanly (minimal-diff rule).
- **Migration path:** a TFE user swaps `required_providers { tfe = { source = "vhco-pro/stackweaver" } }`
  — the provider serves the `tfe_*` types, so existing `resource "tfe_*"` blocks keep working. Later
  they rename to `stackweaver_*` via `moved {}` blocks / `terraform state mv`. Documented in a
  migration guide.

## 3. Strip the unsupported (`dropped`) resources

The `dropped` set = every upstream resource **not** in the spec matrix's implemented list (policy /
Sentinel / OPA, Stacks, no-code modules, admin/enterprise/SAML/SMTP/retention, `oauth_client`,
`ssh_key`, provider-set, etc.). Strip = **remove them from the SDKv2 maps and the framework
factory slices only** (the registration seam) — **keep the `resource_tfe_*.go` files** so upstream
changes to them still merge cleanly; unregistered code is dead, not deleted. The exact list is
enumerated at bootstrap and recorded `dropped` in the matrix. Partial/blocked rows
(`tfe_organization`, `tfe_team_member`, `tfe_github_app_installation`, `tfe_registry_module`) are
also unregistered for v0.1 until their backing API is green.

## 4. Native client (`internal/stackweaver/`)

Created at bootstrap; **the real code-heavy work of the provider** (the 17 native resources + 9 data
sources are Stackweaver-only — no upstream code to fork). House style mirrors `go-tfe` services (a
client holding an `*http.Client` + base URL + token; one file per resource family with typed
`List/Create/Read/Update/Delete` methods). **Critical: the native API envelope is mixed** — most
Ansible resources are JSON:API (`data.attributes`, snake- or dash-cased), a few are plain JSON
(`ansible_config`, notification template/attachment, the VCS listing + playbook-file discovery
endpoints). The client provides **both** a JSON:API codec and a plain-JSON codec; each service method
declares which it uses, per that resource's spec wire contract. Bootstrap ships the client scaffold +
the `ansible_playbook` reference service; the native waves (below) add the rest.

Several native resources have **backing gaps** (model fields not wired in the API:
`credential.vault_id`, `job_template.galaxy_requirements`, `workflow.survey_spec`, some
`workflow_node` targets, `ansible_job` non-`extra_vars` overrides). Those fields are omitted from the
shipped resource and routed to `/tfe-compat` as small backend follow-ups — never faked in the client.

## 5. Fixtures generated from acceptance criteria (the traceability link)

Each `spec/**/<name>.md` **Acceptance criteria** section is the source of that resource's test. The
implement pipeline turns it into a fixture under `test/fixtures/<name>/`:

- The **config** (`main.tf`) instantiates the resource (and any dependency named in the spec's wire
  contract) with the attributes the criteria reference.
- The **assertions** come one-to-one from the criteria bullets, mapped onto the existing
  [`scripts/tfe-compat/`](../../scripts/tfe-compat/README.md) contracts: create → round-trip →
  `terraform plan` shows no drift → destroy → read 404, plus each resource-specific criterion
  (write-only field absent from state, ForceNew recreate, set add/remove, data-source `id` match).
- A criterion with no generated assertion is a **generator bug**, not a pass (requirement→test
  traceability). Data-source fixtures respect the documented Optional-not-Computed plan-null quirk
  (assert the computed `id`).

## 6. Acceptance harness (`dev_overrides`)

`~/.terraformrc` `dev_overrides` → the `go build` binary; fixtures point at the dev stack
(`TFE_HOSTNAME`/`TFE_TOKEN`, org `dev-test`). `dev_overrides` skips `terraform init` (expected). This
is the **real gate** — a resource ships only when its fixture is green here. Upstream `TestAcc*` stay
in the tree as the backport regression net (they need a TFE-shaped backend, i.e. our dev stack).

## 7. CI / release (public repo → free minutes)

Copy `vhco-pro/terraform-provider-garage`'s `.goreleaser.yml` + `.github/workflows/release.yml`.
GPG signing reuses the existing VH&Co key (promote `garage`'s `GPG_PRIVATE_KEY`/`GPG_PASSPHRASE`
repo-secrets to a `vhco-pro` **org secret**) — deferred until first release. **CI honesty:** build +
`go vet` + lint + `go test ./...` (unit) run in CI for free; **full acceptance needs a running
Stackweaver stack, so it runs locally/dev via `dev_overrides`, not in CI** (same posture as the
tfe-compat E2E gate). Documented, not hidden.

## 8. What the automated `implement` pipeline does per resource

Given this plan, each task (see `plan/tasks.md`) is: register the `stackweaver_*` + `tfe_*` names →
`go build` → generate the fixture from the spec criteria → `dev_overrides` acceptance
(**loop-until-green, bounded**; persistent failure ⇒ surface `failed`, never fake done) →
runtime-verify the spec's runtime criterion → docs page + example → one PR (auto-merge on green
optional, decided at bootstrap). The provider-plumbing task (§1–4) runs **once, first**, as the
foundation every resource task builds on.

## Risks specific to the HOW

| Risk | Mitigation |
|---|---|
| Framework alias wrapper misses a delegated method | one generic wrapper embeds the real resource; a compile check + one acceptance run per framework resource proves delegation |
| Stripping a resource breaks a shared helper it's the only caller of | keep files (unregister only); compile after the strip |
| Acceptance can't run in CI (no stack) | dev_overrides local gate is the source of truth; CI does build/unit; documented |
| `tfe_*` alias collides if user also installs hashicorp/tfe | migration guide: use one provider; `moved` to `stackweaver_*` ends the ambiguity |
| Backport conflict on the two edited registration lists | those lists are the accepted seam; resolve there, resource files stay clean |
