---
layout: "stackweaver"
subcategory: "Ansible"
page_title: "Stackweaver: stackweaver_ansible_group"
description: |-
  Manages a group within an Ansible inventory.
---

# stackweaver_ansible_group

Manages a group within a `stackweaver_ansible_inventory`: a named grouping of
hosts with group-level variables, optionally nested under a parent group.

This is a native Stackweaver resource with no Terraform Enterprise equivalent.

## Example Usage

Basic usage:

```hcl
resource "stackweaver_ansible_inventory" "example" {
  organization = "my-org"
  name         = "production"
  type         = "static"
}

resource "stackweaver_ansible_group" "webservers" {
  inventory_id = stackweaver_ansible_inventory.example.id
  name         = "webservers"
  description  = "Front-end web tier"
  variables = {
    http_port = "8080"
  }
}
```

A nested group under a parent:

```hcl
resource "stackweaver_ansible_group" "nginx" {
  inventory_id = stackweaver_ansible_inventory.example.id
  name         = "nginx"
  parent_id    = stackweaver_ansible_group.webservers.id
}
```

## Argument Reference

The following arguments are supported:

* `inventory_id` - (Required) ID of the owning inventory. Changing this forces a
  new group to be created.
* `name` - (Required) Name of the group, unique within the inventory.
* `description` - (Optional) Human-readable description of the group.
* `variables` - (Optional) Map of group-specific variables. May carry secret
  values (not encrypted at rest).
* `parent_id` - (Optional) ID of the parent group for nested groups. Note: the
  backend cannot detach an existing parent in place, so clearing `parent_id`
  after it has been set is not supported.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The group ID.
* `source_id` - Dynamic-source owner. Null for a manually managed group.
* `created_at` - Timestamp the group was created.
* `updated_at` - Timestamp the group was last updated.

## Import

Ansible groups can be imported using their ID. For example:

```shell
terraform import stackweaver_ansible_group.webservers 9c0d1e2f-3a4b-5c6d-7e8f-9a0b1c2d3e4f
```
