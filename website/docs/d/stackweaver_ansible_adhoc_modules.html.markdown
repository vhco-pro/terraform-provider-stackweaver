---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_ansible_adhoc_modules"
subcategory: "Ansible"
description: |-
  Reads the effective ad hoc module allowlist for an organization.
---

# stackweaver_ansible_adhoc_modules (Data Source)

Use this data source to read the effective ad hoc module allowlist for an organization: the modules an
AWX-style Run Command execution is permitted to use. The result is either the organization's configured
list or the built-in AWX default.

This is a native Stackweaver data source with no `terraform-provider-tfe` equivalent.

## Example Usage

```hcl
data "stackweaver_ansible_adhoc_modules" "allowed" {
  organization = "my-org-name"
}

output "adhoc_modules" {
  value = data.stackweaver_ansible_adhoc_modules.allowed.modules
}
```

## Argument Reference

The following arguments are supported:

* `organization` - (Optional) Name of the organization. Defaults to the provider's organization.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The organization's ID.
* `modules` - The effective ad hoc module allowlist.
