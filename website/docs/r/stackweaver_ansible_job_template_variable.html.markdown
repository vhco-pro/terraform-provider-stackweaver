---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_ansible_job_template_variable"
subcategory: "Ansible"
description: |-
  Manages variables scoped to an Ansible job template.
---

# stackweaver_ansible_job_template_variable

Native Stackweaver resource — there is no `terraform-provider-tfe` equivalent.

Provides a single variable scoped to one Ansible job template (the AWX/TFE analogue of a
workspace variable). Each variable is a key/value pair with a category (`env` or
`terraform`) and an optional `sensitive` flag. When `sensitive` is `true`, the value is
write-only: the server encrypts it at rest and masks it on read, so the configured value
is retained in state rather than re-read from the API.

## Example Usage

Basic usage:

```hcl
resource "stackweaver_ansible_job_template" "deploy" {
  organization = "my-org-name"
  name         = "deploy-web"
  playbook_id  = stackweaver_ansible_playbook.site.id
  inventory_id = stackweaver_ansible_inventory.production.id
}

resource "stackweaver_ansible_job_template_variable" "app_version" {
  job_template_id = stackweaver_ansible_job_template.deploy.id
  key             = "app_version"
  value           = "1.2.3"
  category        = "env"
}
```

A sensitive variable:

```hcl
resource "stackweaver_ansible_job_template_variable" "api_token" {
  job_template_id = stackweaver_ansible_job_template.deploy.id
  key             = "api_token"
  value           = var.api_token
  description     = "API token injected into the run"
  sensitive       = true
}
```

## Argument Reference

The following arguments are supported:

* `job_template_id` - (Required) ID of the owning job template. Changing this forces a new variable.
* `key` - (Required) Variable key, unique within the template.
* `value` - (Required) Variable value. When `sensitive` is `true`, the value is write-only and is retained in state because the API masks it on read.
* `description` - (Optional) Human-readable description of the variable.
* `category` - (Optional) Variable category, either `env` (default) or `terraform`.
* `hcl` - (Optional) Whether the value is HCL. Carried for TFE compatibility; not used for Ansible execution. Defaults to `false`.
* `sensitive` - (Optional) Whether the value is sensitive. When `true`, the server encrypts it at rest and masks it on read. Defaults to `false`.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The variable ID (`var-…` format).

## Import

Because there is no single-variable read endpoint, import requires both the job template ID
and the variable ID, in the form `<job_template_id>/<variable_id>`. For example:

```shell
terraform import stackweaver_ansible_job_template_variable.app_version 5e7f8a90-1b2c-3d4e-5f60-7a8b9c0d1e2f/var-4pQnRvXyZ7bT8cWd
```
