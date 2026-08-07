---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_ansible_job_template"
subcategory: "Ansible"
description: |-
  Manages Ansible job templates.
---

# stackweaver_ansible_job_template

Native Stackweaver resource — there is no `terraform-provider-tfe` equivalent.

Provides an Ansible job template: the central, AWX-style reusable run configuration.
A job template binds a playbook, an inventory, and (optionally) a credential together
with the execution knobs (verbosity, forks, tags, become, diff, timeout, concurrency,
slicing) and the launch triggers (schedule, callbacks, webhook) that a launched job
inherits.

Jobs are created *from* a template by a separate launch action; that launch is not part
of this resource's lifecycle.

Related resources:

* Scope variables to a template with [`stackweaver_ansible_job_template_variable`](stackweaver_ansible_job_template_variable.html).
* Attach additional credentials (AWX one-credential-per-type) with [`stackweaver_ansible_job_template_credential`](stackweaver_ansible_job_template_credential.html). The `credential_id` argument below only tracks the single legacy machine credential.

## Example Usage

Basic usage:

```hcl
resource "stackweaver_ansible_job_template" "deploy" {
  organization = "my-org-name"
  name         = "deploy-web"
  playbook_id  = stackweaver_ansible_playbook.site.id
  inventory_id = stackweaver_ansible_inventory.production.id
}
```

With execution knobs and a credential:

```hcl
resource "stackweaver_ansible_job_template" "deploy" {
  organization   = "my-org-name"
  name           = "deploy-web"
  description     = "Deploy the web tier"
  playbook_id    = stackweaver_ansible_playbook.site.id
  inventory_id   = stackweaver_ansible_inventory.production.id
  credential_id  = stackweaver_ansible_credential.ssh.id

  limit          = "web"
  tags           = "deploy"
  verbosity      = 1
  forks          = 10
  become_enabled = true
  diff_mode      = true
  timeout_seconds = 3600

  extra_vars = {
    app_version = "1.2.3"
  }
}
```

## Argument Reference

The following arguments are supported:

* `organization` - (Required) Name of the organization. Changing this forces a new job template.
* `name` - (Required) Name of the job template, unique within the project.
* `playbook_id` - (Required) ID of the playbook to run.
* `inventory_id` - (Required) ID of the inventory to run against.
* `project_id` - (Optional) ID of the owning project. Defaults to the organization's first project when omitted. Changing this forces a new job template.
* `credential_id` - (Optional) ID of the legacy single machine credential. Multi-credential attachment is managed by [`stackweaver_ansible_job_template_credential`](stackweaver_ansible_job_template_credential.html).
* `agent_pool_id` - (Optional) ID of the agent pool to run on. Must belong to the same organization.
* `description` - (Optional) Human-readable description of the job template.
* `extra_vars` - (Optional) A map of extra variables passed to the playbook run. Defaults to an empty map.
* `limit` - (Optional) Host pattern limit (`--limit`).
* `tags` - (Optional) Only run plays and tasks tagged with these values (`--tags`).
* `skip_tags` - (Optional) Skip plays and tasks tagged with these values (`--skip-tags`).
* `verbosity` - (Optional) Ansible verbosity level, 0-4. Defaults to `0`.
* `forks` - (Optional) Number of parallel forks. Defaults to `5`; `0` is coerced to `5`.
* `become_enabled` - (Optional) Enable privilege escalation (become / sudo). Defaults to `false`.
* `diff_mode` - (Optional) Run in diff mode (`--diff`). Defaults to `false`.
* `enabled` - (Optional) Whether the template is enabled. Defaults to `true`.
* `timeout_seconds` - (Optional) Run timeout in seconds. `0` means no timeout. Defaults to `0`.
* `allow_simultaneous` - (Optional) Allow simultaneous runs of this template (AWX concurrent-run semantics). Defaults to `false`.
* `retention_days` - (Optional) Per-template job retention override. Omit to inherit the organization setting; `0` keeps jobs forever.
* `job_slice_count` - (Optional) Number of job slices. Defaults to `1`; a value greater than `1` slices the launch.
* `schedule_enabled` - (Optional) Whether the template's schedule is enabled. Defaults to `false`.
* `schedule_cron` - (Optional) Cron expression for the template's schedule.
* `allow_callbacks` - (Optional) Enable provisioning callbacks. This argument is update-only: it is not accepted on create and is applied by a follow-up update.
* `launch_on_webhook` - (Optional) Launch on webhook. This argument is update-only: it is not accepted on create and is applied by a follow-up update.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The job template ID.
* `host_config_key` - Host config key for provisioning callbacks. Read-only.
* `created_at` - Timestamp when the job template was created.
* `updated_at` - Timestamp when the job template was last updated.

## Import

Ansible job templates can be imported; use the job template ID as the import ID. For example:

```shell
terraform import stackweaver_ansible_job_template.deploy 5e7f8a90-1b2c-3d4e-5f60-7a8b9c0d1e2f
```
