---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_ansible_notification_template"
subcategory: "Ansible"
description: |-
  Manages an org-scoped Ansible notification template (channel).
---

# stackweaver_ansible_notification_template

Provides an Ansible notification template - an org-scoped notification channel (AWX-style) that can be
attached to job templates and workflows through a `stackweaver_ansible_notification_attachment`. The
delivery channel is a `webhook`, `email`, or `teams` channel; channel-specific non-secret settings live in
`config`, and the single sensitive value (webhook basic-auth password / SMTP password) is a **write-only**
`secret`.

This is a native Stackweaver resource with no `terraform-provider-tfe` equivalent. The organization is
taken from the provider configuration; it is not an argument on this resource.

## Example Usage

Basic usage - a webhook notification template:

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
    headers = {
      "X-Source" = "stackweaver"
    }
    username        = "ci"
    skip_tls_verify = false
  })

  # Write-only: sent on create/update, never read back from the API.
  secret = var.webhook_password
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Name of the notification template, unique within the organization.
* `type` - (Required) Delivery channel: `webhook`, `email`, or `teams`. Determines the expected `config`
  keys. Changing this forces a new template.
* `description` - (Optional) A human-readable description of the template.
* `config` - (Optional) Channel-specific non-secret settings as a JSON object string (use `jsonencode`).
  The expected keys depend on `type`:
    * `webhook` - `url`, `method`, `headers` (map), `username`, `skip_tls_verify` (the password goes in
      `secret`).
    * `email` - `host`, `port`, `username`, `from`, `to` (list), `use_tls` (the SMTP password goes in
      `secret`).
    * `teams` - `url`.
* `secret` - (Optional, Write-only) The channel's single sensitive value (webhook basic-auth password /
  SMTP password). It is encrypted at rest and sent on create/update, but is **never read back** from the
  API, so it does not appear in state refreshed from the server and Terraform will not report drift on it.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The notification template ID.

## Import

Stackweaver has no GET-by-id endpoint for notification templates: the resource `Read` resolves the
template by listing the organization's templates and filtering client-side by `id`. A template deleted
out-of-band disappears from the list and is dropped from state on the next refresh. Import by ID still
works, but note that the write-only `secret` cannot be recovered on import and must be re-supplied in
configuration.

Ansible notification templates can be imported using their ID. For example:

```shell
terraform import stackweaver_ansible_notification_template.webhook 3f1a9c2e-8b7d-4e6a-9f10-2c5d7e8a1b34
```
