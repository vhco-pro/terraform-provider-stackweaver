---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_runners"
subcategory: "VCS"
description: |-
  Lists the self-hosted runner fleet in an organization, with a total/online/offline summary.
---

# stackweaver_runners (Data Source)

Use this data source to list the self-hosted runner fleet in a Stackweaver organization, optionally
filtered by agent pool, runner type, or reported status, together with a summary of how many runners
are total, online, and offline.

This is a native Stackweaver data source with no `terraform-provider-tfe` equivalent. There is
deliberately no `stackweaver_runner` resource: runners self-register via the agent API and report their
own metadata and health, so Terraform never creates or owns a runner. This data source is the discovery
counterpart to the Runners page in the UI.

## Example Usage

```hcl
data "stackweaver_runners" "terraform" {
  organization = "my-org-name"
  runner_type  = "terraform"
}

output "online_runners" {
  value = data.stackweaver_runners.terraform.stats.online
}
```

## Argument Reference

The following arguments are supported:

* `organization` - (Optional) Name of the organization. Defaults to the provider's default organization.
* `agent_pool_id` - (Optional) Filter the fleet to a single agent pool.
* `runner_type` - (Optional) Filter the fleet by runner type. One of `terraform`, `ansible`, or `combined`.
* `status` - (Optional) Filter the fleet by reported status. One of `online`, `offline`, `busy`, or `error`.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - Synthetic identifier — the organization name.
* `runners` - The runner fleet matching the filters. Each element documented below.
* `stats` - Fleet summary counts, documented below.

The `runners` block contains:

* `id` - Server-assigned runner ID.
* `name` - Runner name, unique within the organization.
* `agent_pool_id` - ID of the agent pool the runner belongs to.
* `runner_type` - Runner type.
* `status` - Reported health status.
* `hostname` - Agent-reported hostname.
* `os_type` - Agent-reported OS type.
* `agent_version` - Agent-reported version.
* `labels` - Agent-reported labels.
* `terraform_version` - Terraform capability version.
* `ansible_version` - Ansible capability version.
* `last_heartbeat_at` - Timestamp of the last heartbeat (empty when never seen).

The `stats` block contains:

* `total` - Total number of registered runners.
* `online` - Number of online runners.
* `offline` - Number of offline runners.
