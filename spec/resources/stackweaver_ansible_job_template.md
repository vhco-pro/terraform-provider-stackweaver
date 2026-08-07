<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_job_template
tfe_alias: n/a
kind: resource
family: ansible
origin: native
backing_api: implemented
client_approach: native-client
status: spec'd
upstream_file: n/a (native - no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + core/models/ansible_playbook.go)
---
# stackweaver_ansible_job_template

**Native resource - no TFE equivalent.** The central, AWX-style reusable Ansible run configuration.
A job template binds a playbook, an inventory, and (optionally) a credential together with the
execution knobs (verbosity, forks, tags, become, diff, timeout, concurrency, slicing) and the
launch triggers (schedule, webhook, provisioning callbacks) that a launched job inherits. Model:
`core/models/ansible_playbook.go` (`AnsibleJobTemplate`, ~76). Jobs are created *from* a template by
the separate `POST /ansible/job-templates/:id/launch` ephemeral action (see Divergences) - that
launch is **not** part of this resource's CRUD.

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `AnsibleJobTemplates`
service (List/Create/Read/Update/Delete) calling the Stackweaver Ansible API over HTTP.

**Envelope is JSON:API, not plain JSON.** Unlike the `stackweaver_ansible_playbook` exemplar (which
uses plain-`json`-tagged bodies), the job-template handler
(`backend/internal/api/v2/handlers/ansible/playbooks.go`) speaks TFE-style JSON:API:
`{data:{type:"ansible-job-templates", attributes:{...hyphenated...}, relationships:{...}}}`.
Attribute keys are hyphenated (`extra-vars`, `become-enabled`, `job-slice-count`), and the
playbook/inventory/credential/agent-pool/project links live under `relationships`. The native
client must marshal that shape (confirm exact keys against `formatJobTemplateResponse` and
`CreateJobTemplateRequest` at implement time).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string (uuid) | Computed | - | - | no | server-assigned |
| `organization` | string | Required | yes | - | no | create is org-scoped (`POST /organizations/:name/...`); not echoed in attributes |
| `project_id` | string (uuid) | Optional+Computed | yes | first project in org | no | `relationships.project`; defaults to org's first project when omitted; `(project_id,name)` unique |
| `playbook_id` | string (uuid) | Required | no | - | no | `relationships.playbook` |
| `inventory_id` | string (uuid) | Required | no | - | no | `relationships.inventory` |
| `name` | string | Required | no | - | no | unique within the project |
| `description` | string | Optional | no | `""` | no | |
| `extra_vars` | map/jsonb | Optional+Computed | no | `{}` | no | `extra-vars`; passed to the playbook |
| `limit` | string | Optional | no | `""` | no | host pattern limit |
| `tags` | string | Optional | no | `""` | no | `--tags` |
| `skip_tags` | string | Optional | no | `""` | no | `skip-tags` → `--skip-tags` |
| `verbosity` | int | Optional | no | `0` | no | 0–4 |
| `forks` | int | Optional+Computed | no | `5` | no | 0 is coerced to 5 on create |
| `credential_id` | string (uuid) | Optional | no | - | no | `relationships.credential`; the legacy single machine credential (multi-cred is `stackweaver_ansible_job_template_credential`) |
| `agent_pool_id` | string (uuid) | Optional | no | - | no | `relationships.agent-pool`; validated in-org |
| `become_enabled` | bool | Optional | no | `false` | no | `become-enabled`; sudo escalation |
| `diff_mode` | bool | Optional | no | `false` | no | `diff-mode` |
| `enabled` | bool | Optional+Computed | no | `true` | no | inverse of model `Disabled`; omit → stays enabled |
| `timeout_seconds` | int | Optional | no | `0` | no | `timeout-seconds`; 0 = no timeout |
| `allow_simultaneous` | bool | Optional | no | `false` | no | `allow-simultaneous`; AWX concurrent-run semantics |
| `job_slice_count` | int | Optional+Computed | no | `1` | no | `job-slice-count`; normalized (>1 slices the launch) |
| `retention_days` | int | Optional | no | - (inherit org) | no | `retention-days`; nil=inherit, 0=keep forever, negative clears override |
| `schedule_enabled` | bool | Optional | no | `false` | no | `schedule-enabled` |
| `schedule_cron` | string | Optional | no | `""` | no | `schedule-cron`; cron expression |
| `allow_callbacks` | bool | Optional+Computed | no | `false` | no | `allow-callbacks`; **update-only input** (not settable on create) |
| `launch_on_webhook` | bool | Optional+Computed | no | `false` | no | `launch-on-webhook`; **update-only input** (not settable on create) |
| `host_config_key` | string | Computed | - | `""` | no | `host-config-key`; read-only in the API (no create/update input) |
| `created_at` / `updated_at` | string (rfc3339) | Computed | - | - | no | `created-at` / `updated-at` |

`galaxy_requirements` exists on the model but is accepted by neither the create nor the update
request struct - it is **not** a manageable attribute today (flag below).

## Wire contract

- **Create:** `POST /organizations/:name/ansible/job-templates` - JSON:API body; `attributes`:
  `name`, `description`, `extra-vars`, `limit`, `tags`, `skip-tags`, `verbosity`, `forks`,
  `become-enabled`, `diff-mode`, `schedule-enabled`, `schedule-cron`, `enabled` (ptr),
  `timeout-seconds`, `allow-simultaneous`, `retention-days`, `job-slice-count`; `relationships`:
  `project?`, `playbook` (required), `inventory` (required), `credential?`, `agent-pool?`. Returns
  `201` with the full resource. (Create does **not** accept `allow-callbacks`, `launch-on-webhook`,
  `host-config-key`, or `galaxy-requirements`.)
- **Read:** `GET /ansible/job-templates/:id`.
- **Update:** `PATCH /ansible/job-templates/:id` - pointer/optional JSON:API attributes; adds
  `allow-callbacks` and `launch-on-webhook` over the create set. Relationships (playbook,
  inventory, credential, agent-pool) are re-settable. `project_id` is not updatable → ForceNew.
- **Delete:** `DELETE /ansible/job-templates/:id` → `204`.
- **List (read/import helpers):** `GET /organizations/:name/ansible/job-templates`,
  `GET /projects/:id/ansible/job-templates`.
- **JSON:API type:** `ansible-job-templates`. `enabled` is the API projection of the inverted model
  field `Disabled`. `forks:0` is normalized to 5 server-side. `retention-days` is a nullable int.

## Acceptance criteria (these ARE the test)

1. `apply` of `{organization, playbook_id, inventory_id, name}` creates the template; `id`, `name`,
   `playbook_id`, `inventory_id`, `enabled`, `forks`, `verbosity`, `job_slice_count` round-trip into
   state.
2. Re-`plan` after apply shows **no drift** - in particular computed defaults `forks=5`,
   `job_slice_count=1`, `enabled=true`, `extra_vars={}` must settle without a perpetual diff.
3. Omitting `forks` (or setting `0`) settles to `5`; omitting `enabled` settles to `true`;
   omitting `project_id` binds to the org's first project and that id round-trips.
4. Updating in-place applies without recreate for: `description`, `extra_vars`, `verbosity`,
   `forks`, `tags`, `become_enabled`, `timeout_seconds`, `allow_simultaneous`, `allow_callbacks`,
   `launch_on_webhook`, `credential_id`, `agent_pool_id`, `schedule_enabled`/`schedule_cron`.
5. Changing `project_id` (or `organization`) forces recreate.
6. `enabled=false` persists and reads back as `enabled=false` (verifying the `Disabled` inversion is
   not double-flipped).
7. `allow_callbacks`/`launch_on_webhook` set via a follow-up `apply` (update) persist and read back;
   the provider must not expect them to take on create.
8. `host_config_key` is Computed-only - it appears in state from the read but is never sent as input
   and never forces a diff.
9. `destroy` removes it; a subsequent `GET /ansible/job-templates/:id` returns 404.

## Runtime criterion

The job template's runtime effect is that a launch resolves it into a real Ansible job: `POST
/ansible/job-templates/:id/launch` (separate ephemeral action) must produce a job that inherits the
template's playbook, inventory, credential, and execution knobs. Verified: create a template then
launch (or dry-run) it and confirm the launched job carries the configured `limit`/`verbosity`/
`become`/`extra_vars`. Config-with-real-effect, not dead CRUD.

## Docs + example

- Provider docs page: `docs/resources/ansible_job_template.md` - document the create-only vs
  update-only attribute split (`allow_callbacks`/`launch_on_webhook` update-only; `host_config_key`
  computed) and the `enabled` (inverted `Disabled`) semantics.
- Example: `examples/resources/stackweaver_ansible_job_template/resource.tf` - a project + a
  `stackweaver_ansible_playbook` + an inventory, wired into a minimal template.

## Divergences from upstream / TFE

Native resource - no TFE equivalent. Notable Stackweaver specifics:
- **Launch is not CRUD.** `POST /ansible/job-templates/:id/launch` creates a job; it is a separate
  ephemeral action, out of scope for this resource.
- **Create/update attribute asymmetry.** `allow_callbacks` and `launch_on_webhook` are update-only;
  `host_config_key` is read-only; `galaxy_requirements` (present on the model) is accepted by
  **neither** request struct and is therefore unmanageable today - flag for API follow-up if the
  provider is meant to manage Galaxy requirements.
- **`enabled` inverts the model's `Disabled`** field on the wire.
- Multi-credential attachment (AWX "one per type") is a **separate** resource,
  `stackweaver_ansible_job_template_credential`; the `credential_id` attribute here only tracks the
  single legacy machine credential.
