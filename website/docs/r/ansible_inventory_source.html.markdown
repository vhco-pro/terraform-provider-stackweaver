---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_ansible_inventory_source"
subcategory: "Ansible"
description: |-
  Manages a dynamic inventory source on an Ansible inventory.
---

# stackweaver_ansible_inventory_source

Attaches a dynamic inventory source to an existing Ansible inventory. Hosts are discovered from a
cloud provider (AWS EC2, Azure VMs, GCP Compute, VMware vCenter) or a custom script, using an optional
cloud credential and provider-specific `config`. A sync populates the inventory's hosts and groups.

This is a native Stackweaver resource with no `terraform-provider-tfe` equivalent.

The owning `inventory_id` and the `source_type` are immutable - changing either forces a new source.
The `config` argument is a provider-specific JSON string (use `jsonencode`); its keys depend on
`source_type` (for example AWS regions and filters, Azure resource groups, GCP projects and zones, or
VMware connection settings). The server defaults `config` to an empty object and may reorder its keys.

Syncing an inventory source is a Stackweaver action with no Terraform lifecycle analogue; it is
triggered outside of `terraform apply`.

## Example Usage

An AWS source on an inventory, referencing a cloud credential:

```hcl
resource "stackweaver_ansible_credential" "aws" {
  organization          = "my-org-name"
  name                  = "aws-inventory"
  credential_type       = "aws"
  aws_access_key_id     = var.aws_access_key_id
  aws_secret_access_key = var.aws_secret_access_key
}

resource "stackweaver_ansible_inventory_source" "aws" {
  inventory_id  = "c1d2e3f4-5a6b-4c7d-8e9f-0a1b2c3d4e5f"
  name          = "aws-ec2"
  source_type   = "aws"
  credential_id = stackweaver_ansible_credential.aws.id

  config = jsonencode({
    regions = ["eu-west-1"]
  })

  update_on_launch = true
}
```

## Argument Reference

The following arguments are supported:

* `inventory_id` - (Required) ID of the owning inventory. Changing this forces a new source.
* `name` - (Required) Name of the inventory source (1-255 characters).
* `source_type` - (Required) Source type: one of `aws`, `azure`, `gcp`, `vmware`, `custom`. Immutable
  - changing it forces a new source.
* `description` - (Optional) Human-readable description of the source.
* `credential_id` - (Optional) ID of the cloud credential used for discovery. Omit (or set empty) to
  use workload-identity/OIDC.
* `config` - (Optional) Provider-specific configuration as a JSON string (use `jsonencode`). Defaults
  to an empty object.
* `update_on_launch` - (Optional) Sync the source before each job run. Defaults to `true`.
* `update_cache_timeout` - (Optional) Seconds a prior sync stays fresh for update-on-launch (`0` =
  always sync). Defaults to `0`.
* `overwrite` - (Optional) Remove source-owned hosts and groups the provider no longer reports.
  Defaults to `false`.
* `overwrite_vars` - (Optional) Replace host vars wholesale on sync (`false` = merge). Defaults to
  `false`.
* `verbosity` - (Optional) Verbosity 0-4, adds `-v`..`-vvvv` to `ansible-inventory`. Defaults to `0`.
* `group_by_instance_id` - (Optional) Group discovered hosts by instance ID. Defaults to `false`.
* `group_by_region` - (Optional) Group discovered hosts by region. Defaults to `true`.
* `group_by_availability_zone` - (Optional) Group discovered hosts by availability zone. Defaults to
  `false`.
* `group_by_tag` - (Optional) Tag key to group discovered hosts by (for example `Environment`).
* `hostname_var` - (Optional) Which variable becomes the Ansible hostname. Defaults to `public_ip`.
* `instance_filters` - (Optional) JSON array of provider-specific instance filters.
* `sync_schedule` - (Optional) Cron expression for scheduled sync (for example `0 */6 * * *`).
* `enabled` - (Optional) Whether the source is enabled. Disabled sources reject sync. Defaults to
  `true`.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The inventory source ID.
* `status` - Sync status: `never_synced`, `syncing`, `successful`, or `failed`.
* `last_sync_at` - Timestamp of the most recent sync.
* `last_sync_error` - Error from the most recent sync, if any.
* `last_sync_log` - Stderr and warnings captured from the most recent sync.
* `hosts_count` - Number of hosts discovered by the most recent sync.

## Import

Ansible inventory sources can be imported using their ID. For example:

```shell
terraform import stackweaver_ansible_inventory_source.aws 7a8b9c0d-1e2f-4a3b-8c4d-5e6f7a8b9c0d
```
