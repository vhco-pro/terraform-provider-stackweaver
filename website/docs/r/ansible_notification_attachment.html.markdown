---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_ansible_notification_attachment"
subcategory: "Ansible"
description: |-
  Attaches an Ansible notification template to a job template or workflow.
---

# stackweaver_ansible_notification_attachment

Binds a `stackweaver_ansible_notification_template` (channel) to exactly one target — a job template or a
workflow — with per-trigger flags controlling when the channel fires. This is a create/delete-only
relationship: every attribute forces replacement, so there is no in-place update.

This is a native Stackweaver resource with no `terraform-provider-tfe` equivalent. The organization is
taken from the provider configuration; it is not an argument on this resource.

## Example Usage

Attach a template to a job template so it fires on success:

```hcl
provider "stackweaver" {
  hostname     = "stackweaver.example.com"
  token        = var.stackweaver_token
  organization = "my-org"
}

resource "stackweaver_ansible_notification_template" "webhook" {
  name = "ops-webhook"
  type = "webhook"

  config = jsonencode({
    url    = "https://hooks.example.com/ansible"
    method = "POST"
  })
}

resource "stackweaver_ansible_notification_attachment" "deploy_success" {
  notification_template_id = stackweaver_ansible_notification_template.webhook.id
  job_template_id          = stackweaver_ansible_job_template.deploy.id

  on_started = false
  on_success = true
  on_failure = false
}
```

## Argument Reference

The following arguments are supported:

* `notification_template_id` - (Required) ID of the notification template (channel) being attached.
  Changing this forces a new attachment.
* `job_template_id` - (Optional) ID of the job template to attach to. Exactly one of `job_template_id` or
  `workflow_id` must be set. Changing this forces a new attachment.
* `workflow_id` - (Optional) ID of the workflow to attach to. Exactly one of `job_template_id` or
  `workflow_id` must be set. Changing this forces a new attachment.
* `on_started` - (Optional) Fire the channel when the target starts. Defaults to `false`. Changing this
  forces a new attachment.
* `on_success` - (Optional) Fire the channel on success. Defaults to `true`. Changing this forces a new
  attachment.
* `on_failure` - (Optional) Fire the channel on failure. Defaults to `true`. Changing this forces a new
  attachment.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The attachment ID.

## Import

Ansible notification attachments can be imported using their ID. For example:

```shell
terraform import stackweaver_ansible_notification_attachment.deploy_success 8c2f5d1a-4b9e-4c7a-a3d6-1e0f9b2c3d45
```

~> **Note:** Workflow-target attachments have no server-side listing route, so the provider cannot detect
out-of-band drift for them on refresh (their prior state is preserved). Job-template-target attachments are
read back through the job template's notifications and will be removed from state if deleted out of band.
