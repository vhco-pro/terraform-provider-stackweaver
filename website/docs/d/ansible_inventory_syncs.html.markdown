---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_ansible_inventory_syncs"
subcategory: "Ansible"
description: |-
  Lists the sync-run history of one Ansible inventory, newest first.
---

# stackweaver_ansible_inventory_syncs (Data Source)

Use this data source to list the sync-run history (AWX's inventory update jobs) of one Ansible inventory,
newest first. The captured output of each run is intentionally omitted from the listing.

This is a native Stackweaver data source with no `terraform-provider-tfe` equivalent.

## Example Usage

```hcl
data "stackweaver_ansible_inventory_syncs" "history" {
  inventory_id = "inv-0123456789abcdef"
}

output "last_sync_status" {
  value = try(data.stackweaver_ansible_inventory_syncs.history.syncs[0].status, null)
}
```

## Argument Reference

The following arguments are supported:

* `inventory_id` - (Required) ID of the inventory whose sync history to list.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - Set to the requested `inventory_id`.
* `syncs` - Sync runs, newest first. The captured output is intentionally omitted. Each element documented below.

The `syncs` block contains:

* `id` - Sync run ID.
* `status` - Lifecycle status: `pending`, `running`, `successful`, or `failed`.
* `triggered_by` - What started the run: `manual`, `schedule`, `launch`, `workflow`, or `webhook`.
* `hosts_discovered` - Number of hosts discovered by the run.
* `groups_discovered` - Number of groups discovered by the run.
* `source_name` - Name of the dynamic source, when the run is a source sync.
* `error` - Failure detail; empty on success.
* `started_at` - RFC3339 start time; null until the run starts.
* `finished_at` - RFC3339 finish time; null until the run finishes.
* `created_at` - RFC3339 creation time.
