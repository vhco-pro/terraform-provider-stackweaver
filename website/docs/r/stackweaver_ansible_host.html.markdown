---
layout: "stackweaver"
subcategory: "Ansible"
page_title: "Stackweaver: stackweaver_ansible_host"
description: |-
  Manages a single host within an Ansible inventory.
---

# stackweaver_ansible_host

Manages a single host within a `stackweaver_ansible_inventory`: a named target
with an optional distinct hostname or IP, an SSH port, per-host variables, and an
enabled flag.

This is a native Stackweaver resource with no Terraform Enterprise equivalent.

## Example Usage

Basic usage:

```hcl
resource "stackweaver_ansible_inventory" "example" {
  organization = "my-org"
  name         = "production"
  type         = "static"
}

resource "stackweaver_ansible_host" "web" {
  inventory_id = stackweaver_ansible_inventory.example.id
  name         = "web-01"
  hostname     = "10.0.0.10"
  port         = 22
  variables = {
    ansible_user = "deploy"
  }
}
```

## Argument Reference

The following arguments are supported:

* `inventory_id` - (Required) ID of the owning inventory. Changing this forces a
  new host to be created.
* `name` - (Required) Name of the host, unique within the inventory.
* `description` - (Optional) Human-readable description of the host.
* `hostname` - (Optional) Actual hostname or IP if different from `name`.
* `port` - (Optional) SSH port. Defaults to `22`.
* `variables` - (Optional) Map of host-specific variables. May carry secret
  values (not encrypted at rest).
* `enabled` - (Optional) Whether the host is included at run time. Defaults to
  `true`.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The host ID.
* `source_id` - Dynamic-source owner. Null for a manually managed host.
* `created_at` - Timestamp the host was created.
* `updated_at` - Timestamp the host was last updated.

## Import

Ansible hosts can be imported using their ID. For example:

```shell
terraform import stackweaver_ansible_host.web 8b9c0d1e-2f3a-4b5c-6d7e-8f9a0b1c2d3e
```
