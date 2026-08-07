---
layout: "stackweaver"
subcategory: "Ansible"
page_title: "Stackweaver: stackweaver_ansible_inventory"
description: |-
  Manages an Ansible inventory.
---

# stackweaver_ansible_inventory

Manages an Ansible inventory: a named collection of hosts and groups, either
organization- or project-scoped, of one of four types (`static`, `dynamic`,
`vcs`, `constructed`). Hosts (`stackweaver_ansible_host`), groups
(`stackweaver_ansible_group`), and dynamic sources hang off an inventory, and a
sync action refreshes dynamic, VCS, and constructed inventories.

This is a native Stackweaver resource with no Stackweaver equivalent.

## Example Usage

A static, organization-scoped inventory:

```hcl
resource "stackweaver_ansible_inventory" "production" {
  organization = "my-org"
  name         = "production"
  type         = "static"
}
```

A VCS-backed inventory:

```hcl
resource "stackweaver_ansible_inventory" "from_repo" {
  organization      = "my-org"
  name              = "from-repo"
  type              = "vcs"
  vcs_connection_id = "b1c2d3e4-5f6a-7b8c-9d0e-1f2a3b4c5d6e"
  vcs_repository    = "octocat/ansible-inventory"
  vcs_branch        = "main"
  inventory_path    = "inventories/prod/hosts.yml"
}
```

A constructed inventory composed from other inventories:

```hcl
resource "stackweaver_ansible_inventory" "constructed" {
  organization              = "my-org"
  name                      = "constructed"
  type                      = "constructed"
  source_vars               = <<-EOT
    plugin: constructed
    groups:
      webservers: "'web' in inventory_hostname"
  EOT
  constructed_limit         = "webservers"
  constructed_cache_timeout = 3600
  input_inventory_ids = [
    stackweaver_ansible_inventory.production.id,
  ]
}
```

## Argument Reference

The following arguments are supported:

* `organization` - (Required) Name of the owning organization. Changing this
  forces a new inventory to be created.
* `name` - (Required) Name of the inventory, unique within the organization.
* `project_id` - (Optional) ID of the owning project. When null the inventory is
  organization-scoped. Changing this forces a new inventory to be created.
* `description` - (Optional) Human-readable description of the inventory.
* `type` - (Optional) Inventory type: `static` (default), `dynamic`, `vcs`, or
  `constructed`. Changing this forces a new inventory to be created.
* `variables` - (Optional) Map of global inventory variables. May carry secret
  values (not encrypted at rest).
* `source` - (Optional) Plugin config or legacy VCS URL. Deprecated for VCS
  inventories.
* `vcs_connection_id` - (Optional) ID of the GitHub App VCS connection backing a
  `vcs` inventory.
* `vcs_repository` - (Optional) Full name of the repository, in `"owner/repo"`
  form, for a `vcs` inventory.
* `vcs_branch` - (Optional) Branch to pull for a `vcs` inventory. Defaults to
  `main`.
* `inventory_path` - (Optional) Path to the inventory file within the repository
  for a `vcs` inventory.
* `source_vars` - (Optional) For a `constructed` inventory, the YAML
  compose/groups/keyed_groups rules.
* `constructed_limit` - (Optional) For a `constructed` inventory, the host limit
  expression.
* `constructed_cache_timeout` - (Optional) For a `constructed` inventory, the
  rebuild cache TTL in seconds (`0` = always rebuild).
* `input_inventory_ids` - (Optional) For a `constructed` inventory, an ordered
  list of input inventory IDs.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The inventory ID.
* `last_sync_at` - Timestamp of the most recent sync.
* `last_sync_status` - Status of the most recent sync (`syncing`, `successful`,
  or `failed`).
* `last_sync_error` - Error message from the most recent sync, if any.
* `last_sync_hosts_discovered` - Number of hosts discovered during the most
  recent sync.
* `last_sync_log` - `ansible-inventory` stderr and warnings from the most recent
  sync.
* `created_at` - Timestamp the inventory was created.
* `updated_at` - Timestamp the inventory was last updated.

## Import

Ansible inventories can be imported using their ID. For example:

```shell
terraform import stackweaver_ansible_inventory.production 7a8b9c0d-1e2f-3a4b-5c6d-7e8f9a0b1c2d
```

The `organization` and `project_id` values are not recoverable from the API and
remain null after import until set in configuration.
