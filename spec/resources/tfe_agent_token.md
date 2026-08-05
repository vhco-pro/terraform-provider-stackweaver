<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_agent_token
tfe_alias: tfe_agent_token
kind: resource
family: agent-pools
origin: forked
backing_api: implemented
client_approach: go-tfe-clean   # value-level divergence (documented): created-by relation omitted from Read response; wire shape otherwise matches go-tfe, stock client parses it unchanged
status: spec'd
upstream_file: internal/provider/resource_tfe_agent_token.go
go_tfe_type: AgentToken
compat_doc: docs/internal/tfe-compatibility/resources/agent-pools/tfe_agent_token.md
---
# stackweaver_agent_token

A pool-scoped registration credential: the secret an agent presents to register into an agent pool.
A pool may have **many** agent tokens, each identified by a `description`. Stackweaver maps it onto
its existing API-key infrastructure — an agent token is a `Kind=org` api_key carrying the
`org:<org>:runner:register` scope, bound to the pool (`AgentPoolID`) and flagged `IsAgentToken`; the
runner registration handler enforces that a runner presenting it may only join that pool
(`docs/internal/tfe-compatibility/resources/agent-pools/tfe_agent_token.md`).

## Client approach

`go-tfe-clean` **with a documented value-level divergence**. The upstream legacy SDKv2 resource
(`resourceTFEAgentToken` at `internal/provider/resource_tfe_agent_token.go:19`) drives `go-tfe`'s
`AgentTokens.Create`/`Read`/`Delete`, exchanging the stock `authentication-tokens` JSON:API shape.
Stackweaver returns that same wire shape, so stock `go-tfe` parses it unchanged and **no wrapper is
needed**. The only difference is a value-level one: Stackweaver omits the `created-by` relationship
from the response. This is harmless because the provider's Read consumes only `description`
(`resource_tfe_agent_token.go:89`) — it never reads `created-by`. Captured as a divergence note
below, not client code.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | `authentication-tokens` primary id; read/delete are by this id |
| `agent_pool_id` | string | Required | yes | — | no | pool the token registers into; create is pool-scoped |
| `description` | string | Required | yes | — | no | required by the provider; stored as the api_key name, echoed on read |
| `token` | string | Computed | — | — | **yes** | write-only secret; returned **only** on create, never on read |

## Wire contract

- **Create:** `AgentTokens.Create(agent_pool_id, AgentTokenCreateOptions{Description})` →
  `POST /agent-pools/:id/authentication-tokens`. Request attr: `description` (required). Response:
  full `AgentToken` including `token` (the plaintext secret, returned this once only) and `id`.
  Provider sets `id` and stashes `token` into state during create.
- **Read:** `AgentTokens.Read(id)` → `GET /authentication-tokens/:id`. Response attrs: `description`,
  `created-at`, `last-used-at` (no `token`; `created-by` omitted by Stackweaver). Provider consumes
  `description` only. 404 → resource removed from state.
- **Update:** none — no update handler; both `agent_pool_id` and `description` are ForceNew, so any
  change recreates (new token minted, old revoked).
- **Delete:** `AgentTokens.Delete(id)` → `DELETE /authentication-tokens/:id` (revokes the token).
- **JSON:API type:** `authentication-tokens`. `token` is **write-only** (only echoed on create; the
  backend stores just a bcrypt hash). **Divergence:** the `created-by` relationship is omitted from
  the response (stock go-tfe tolerates its absence).

## Acceptance criteria (these ARE the test)

1. `apply` of `{agent_pool_id, description}` creates the token; `id` and `description` round-trip into
   state, and `token` is populated in state from the create response.
2. Re-`plan` after apply shows **no drift** (Read restores `description`; `token` is not re-fetched).
3. `token` is write-only: it is present in state only from the create response and **never** appears
   in / is refreshed by the Read response — a fixture asserting the read payload has no `token` field
   passes.
4. `agent_pool_id` and `description` are both ForceNew — changing either recreates the resource (a new
   `id` and a new `token` value; the previous token is revoked).
5. `destroy` revokes it; a subsequent `AgentTokens.Read(id)` returns 404.
6. Divergence assertion: the create/read response carries **no** `created-by` relationship, and the
   apply still succeeds (provider does not depend on it).

## Runtime criterion

**Runtime effect (the real gate):** an agent authenticates/registers with the pool using the token.
Verified: a runner calls the registration endpoint presenting the agent token, is accepted (200) and
receives a per-runner control-plane token, then heartbeats (200); a register attempt into a
**different** pool with the same token is rejected (403 — pool binding). Not CRUD-only.

## Docs + example

- Provider docs page: `docs/resources/agent_token.md` — arguments (agent_pool_id/description),
  computed sensitive `token` and `id`, and a prominent note that `token` is shown only once at create
  time and must be captured then (it is never retrievable on read).
- Example: `examples/resources/stackweaver_agent_token/resource.tf` — a `stackweaver_agent_pool` plus
  a `stackweaver_agent_token` referencing it, with the token consumed by a runner/output.

## Divergences from upstream / TFE

**Value-level (documented):** the `created-by` relationship is **omitted** from the Stackweaver
response. The wire *shape* is otherwise identical to go-tfe's `authentication-tokens`, and the
provider's Read consumes only `description`, so stock `go-tfe` parses the response unchanged — no
client change; this is a response-content note only. `token` is write-only (returned once on create,
never on read) exactly as in upstream. Compat source:
`docs/internal/tfe-compatibility/resources/agent-pools/tfe_agent_token.md` (Attribute Mapping —
`created-by` row marked *divergent*; Divergences / scope decisions). Stackweaver-extra behavior
(not a TFE divergence in wire terms): the token is a pool-bound `runner:register` api_key, so the
registration handler confines a runner to the token's pool.
