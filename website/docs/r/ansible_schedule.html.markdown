---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_ansible_schedule"
subcategory: "Ansible"
description: |-
  Manages a cron schedule that periodically triggers an Ansible target.
---

# stackweaver_ansible_schedule

Provides an Ansible schedule - a declarative cron schedule that periodically triggers an Ansible target: a
job template, an inventory-source sync, a playbook sync, or a workflow. The `type` selects which target id
is required.

This is a native Stackweaver resource with no `terraform-provider-tfe` equivalent. The organization is
taken from the provider configuration; it is not an argument on this resource.

## Example Usage

A daily job-template schedule:

```hcl
provider "stackweaver" {
  hostname     = "stackweaver.example.com"
  token        = var.stackweaver_token
  organization = "my-org"
}

resource "stackweaver_ansible_schedule" "nightly" {
  name            = "nightly-deploy"
  type            = "job_template"
  job_template_id = stackweaver_ansible_job_template.deploy.id

  cron_expression = "0 2 * * *"
  timezone        = "America/New_York"

  config = jsonencode({
    extra_vars = {
      environment = "staging"
    }
  })
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Display name of the schedule.
* `type` - (Required) What the schedule triggers: `job_template`, `inventory_source`, `playbook_sync`, or
  `workflow`. Determines which target id is required. Changing this forces a new schedule.
* `cron_expression` - (Required) Standard 5-field cron expression (`minute hour day month weekday`).
* `job_template_id` - (Optional) Target job template (set iff `type = job_template`). Changing this forces
  a new schedule.
* `inventory_source_id` - (Optional) Target inventory source (set iff `type = inventory_source`). Changing
  this forces a new schedule.
* `playbook_id` - (Optional) Target playbook (set iff `type = playbook_sync`). Changing this forces a new
  schedule.
* `workflow_id` - (Optional) Target workflow (set iff `type = workflow`). Changing this forces a new
  schedule.
* `description` - (Optional) A human-readable description of the schedule.
* `timezone` - (Optional) IANA timezone for the cron expression (e.g. `America/New_York`). Defaults to
  `UTC`.
* `start_date_time` - (Optional) RFC3339 instant after which the schedule becomes active. Changing this
  forces a new schedule (not updatable in place).
* `end_date_time` - (Optional) RFC3339 instant after which the schedule stops. Changing this forces a new
  schedule (not updatable in place).
* `config` - (Optional) Extra configuration as a JSON object string (use `jsonencode`), e.g. an
  `extra_vars` override for job-template schedules.

Exactly one of `job_template_id`, `inventory_source_id`, `playbook_id`, or `workflow_id` must be set, and
it must match `type`.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The schedule ID.
* `status` - Whether the schedule is enabled or disabled. Managed server-side (via the enable/disable
  actions); read-only here.
* `next_run_at` - Next calculated run time.
* `last_run_at` - Most recent execution time.
* `last_run_status` - Status of the most recent execution.
* `last_job_id` - ID of the most recent job the schedule created.
* `run_count` - Total number of executions.

## Import

Ansible schedules can be imported using their ID. For example:

```shell
terraform import stackweaver_ansible_schedule.nightly 5d3b7f9c-2a1e-4d8b-9c0f-6e7a8b1c2d34
```
