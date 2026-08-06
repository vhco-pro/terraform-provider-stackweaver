---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_ansible_job"
subcategory: "Ansible"
description: |-
  Launches an Ansible job from a job template (a lifecycle trigger).
---

# stackweaver_ansible_job

Launches an Ansible job from a job template. This is a native Stackweaver resource with no
`terraform-provider-tfe` equivalent; it is modeled on `tfe_workspace_run`.

~> **Important:** This resource is a **launch trigger, not reconciled configuration.** Creating it has a
**side effect** — it launches a job on Stackweaver and (by default) waits for that job to reach a terminal
status. It records a single point-in-time execution; it does not manage the job's ongoing state. All launch
inputs are `ForceNew`, so changing them **launches a new job** (replacement) rather than editing the
existing one, and a no-op re-plan does **not** re-launch. Destroying the resource removes it from state
without undoing the run — a completed job is immutable history.

## Example Usage

Launch a job from a template:

```hcl
provider "stackweaver" {
  hostname     = "stackweaver.example.com"
  token        = var.stackweaver_token
  organization = "my-org"
}

resource "stackweaver_ansible_job" "deploy" {
  job_template_id = stackweaver_ansible_job_template.deploy.id

  extra_vars = jsonencode({
    environment = "production"
    version     = "1.4.2"
  })

  wait_for_completion = true
}
```

## Argument Reference

The following arguments are supported:

* `job_template_id` - (Required) Job template to launch from. A new value performs a new launch (forces
  replacement).
* `extra_vars` - (Optional) Extra-vars override as a JSON object string (use `jsonencode`). Changing it
  performs a new launch (forces replacement). Note: on the template-launch endpoint only `extra_vars` is
  honored today; limit/tags/inventory overrides are not yet backed.
* `wait_for_completion` - (Optional) Whether to poll until the job reaches a terminal status. Defaults to
  `true`. When `false`, apply returns immediately with a pending/running status.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The launched job's ID.
* `status` - Job status: `successful`, `failed`, `canceled`, or `error` after waiting; `pending`/`running`
  when not waiting.
* `started_at` - When the job started (RFC3339).
* `finished_at` - When the job finished (RFC3339).
* `exit_code` - The `ansible-playbook` exit code.
* `hosts_ok` - Number of hosts with ok results.
* `hosts_changed` - Number of hosts with changed results.
* `hosts_failed` - Number of hosts with failed results.
* `hosts_unreachable` - Number of unreachable hosts.

## Import

Ansible jobs can be imported using their ID. For example:

```shell
terraform import stackweaver_ansible_job.deploy 9a4c1e7b-3d6f-4a2e-8b5c-0f1d2e3a4b56
```
