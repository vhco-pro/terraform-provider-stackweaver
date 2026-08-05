<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_runners
tfe_alias: n/a
kind: data-source
family: runners
origin: native
backing_api: implemented
client_approach: native-client
status: spec'd
upstream_file: n/a (native — no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + core/models/runner.go)
---
# stackweaver_runners

**Native data source — no TFE equivalent.** Read-only observability view over the self-hosted runner
fleet in an organization. Runners self-register via the agent API and report their own metadata and
health; there is deliberately no `stackweaver_runner` *resource*, because Terraform does not create or
own a runner. This data source lets a config list the fleet (optionally filtered by agent pool, type,
or status) and read a summary of how many are total / online / offline. It is the discovery counterpart
to the Runners page in the UI. Model: `core/models/runner.go` (`Runner`).

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `Runners` service with a
`List` (fleet) method and a `Stats` (summary) method calling the Stackweaver runners API over HTTP.
Only reads — no create/update/delete of runners from the provider.

**Envelope note:** unlike the VCS native data sources (plain JSON arrays), the runners endpoints emit a
JSON:API-shaped body — `data` is an array of `{ id, type: "runners", attributes, relationships }`
objects whose attribute keys are **dash-cased** (`agent-pool-id`, `runner-type`, `os-type`,
`agent-version`, `max-concurrent-jobs`, `last-heartbeat-at`, …), with `meta.pagination`
(`current-page`, `page-size`, `total-count`, `total-pages`). The stats endpoint returns a single
`{ type: "runner-stats", attributes: { total, online, offline } }`. The native client must unmarshal
this dash-cased JSON:API shape, not the plain `json:"..."` snake_case tags on the Go model. Confirm the
exact keys against `buildRunnerResponse` in `backend/internal/api/v2/handlers/runners.go` at implement
time.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `organization` | string | Optional | — | provider default | no | org name; falls back to the provider's default org |
| `agent_pool_id` | string (uuid) | Optional | — | — | no | filter → `filter[agent_pool_id]` |
| `runner_type` | string | Optional | — | — | no | filter → `filter[runner_type]`; `terraform` \| `ansible` \| `combined` |
| `status` | string | Optional | — | — | no | filter → `filter[status]`; `online` \| `offline` \| `busy` \| `error` |
| `id` | string | Computed | — | — | no | synthetic id = organization name |
| `runners` | list(object) | Computed | — | — | no | the fleet; each object below |
| `runners[].id` | string (uuid) | Computed | — | — | no | server-assigned runner id |
| `runners[].name` | string | Computed | — | — | no | unique within the org |
| `runners[].agent_pool_id` | string (uuid) | Computed | — | — | no | `attributes.agent-pool-id` |
| `runners[].runner_type` | string | Computed | — | — | no | `attributes.runner-type` |
| `runners[].status` | string | Computed | — | — | no | reported health |
| `runners[].hostname` | string | Computed | — | — | no | agent-reported |
| `runners[].os_type` | string | Computed | — | — | no | agent-reported |
| `runners[].agent_version` | string | Computed | — | — | no | agent-reported |
| `runners[].labels` | list(string) | Computed | — | — | no | agent-reported |
| `runners[].terraform_version` / `runners[].ansible_version` | string | Computed | — | — | no | capabilities |
| `runners[].last_heartbeat_at` | string (RFC3339) | Computed | — | — | no | null when never seen |
| `stats` | object | Computed | — | — | no | `{ total, online, offline }` summary |

## Wire contract

- **Read (list):** `Runners.List(org, opts)` → `GET /organizations/:name/runners`. Optional filters map
  to query params `filter[agent_pool_id]`, `filter[runner_type]`, `filter[status]` (also `q`, `sort`,
  `page[size]`, `page[number]` exist server-side). Paginate through all pages via `meta.pagination`
  (`total-pages`), accumulating each `data[]` entry into `runners`.
- **Read (summary):** `Runners.Stats(org)` → `GET /organizations/:name/runners/stats`. Returns
  `{ type: "runner-stats", attributes: { total, online, offline } }` → populate `stats`.
- **Per-runner:** `GET /runners/:id` (`GetByID`) exists but this data source does not need it; the list
  endpoint already returns full per-runner attributes. Document it as available, do not call it.
- No create/update/delete — data source. Runners self-register and self-report; the provider never
  mutates the fleet.
- **Envelope:** JSON:API-shaped, dash-cased attribute keys (see Client approach). Native client owns
  unmarshalling. `PATCH`/`DELETE /runners/:id` exist server-side but are intentionally out of scope.

## Acceptance criteria (these ARE the test)

1. `apply` of `{ organization = <dev org> }` reads the fleet and succeeds; `id` is set to the org name.
2. `stats.total` equals `stats.online + stats.offline`, and `length(runners)` on an unfiltered read
   equals `stats.total` (single-page dev fleet), i.e. the list and the summary agree.
3. Each `runners[]` object round-trips `id`, `name`, `agent_pool_id`, `runner_type`, and `status`; the
   `id` is a non-empty uuid.
4. Filtering by `runner_type` (or `status`) returns only runners matching that value — assert every
   returned `runners[].runner_type == var.runner_type`.
5. Filtering by an `agent_pool_id` that has no runners returns an empty `runners` list and `apply` still
   succeeds (empty is not an error for this observability view).
6. **Fleet-may-be-empty caveat:** the dev stack can legitimately have zero registered runners. The test
   fixture must therefore either (a) register a runner via the agent API first, or (b) assert only the
   invariant outputs — `id` set, `stats` present, `total == online + offline`, `runners` is a list —
   rather than a hard-coded non-zero count. Prefer (a) if a registration helper is available.

## Runtime criterion

`CRUD-only` (read-only). No runtime side effect: it observes a fleet that registers and heartbeats out
of band. Its correctness criterion is agreement between the list and the stats summary (criterion 2)
and that filters narrow the set (criterion 4).

## Docs + example

- Provider docs page: `docs/data-sources/runners.md` — arguments (`organization`, `agent_pool_id`,
  `runner_type`, `status`), computed `runners` list (with nested object attributes) and `stats`.
- Example: `examples/data-sources/stackweaver_runners/data-source.tf` — list a fleet filtered to
  `runner_type = "terraform"` and output `data.stackweaver_runners.x.stats.online`.

## Divergences from upstream / TFE

Native data source — no TFE equivalent. No corresponding resource by design (runners self-register).
Envelope is JSON:API-shaped with dash-cased keys, diverging from the plain-JSON VCS native data sources
in this family — the native client must special-case it. `PATCH`/`DELETE /runners/:id` are out of scope.
