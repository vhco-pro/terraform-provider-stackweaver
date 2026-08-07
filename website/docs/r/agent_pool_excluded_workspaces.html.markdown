---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_agent_pool_excluded_workspaces"
description: |-
  Manages excluded workspaces on agent pools
---

# stackweaver_agent_pool_excluded_workspaces

Adds and removes excluded workspaces on an agent pool.

~> **NOTE:** This resource requires using the provider with Stackweaver and a Stackweaver
for Business account.
[Learn more about Stackweaver pricing here](https://stackweaver.io/pricing).

## Example Usage

```hcl
resource "stackweaver_organization" "test-organization" {
  name  = "my-org-name"
  email = "admin@company.com"
}

// Ensure workspace and agent pool are create first
resource "stackweaver_workspace" "test-workspace" {
  name         = "my-workspace-name"
  organization = stackweaver_organization.test-organization.name
}

resource "stackweaver_agent_pool" "test-agent-pool" {
  name                = "my-agent-pool-name"
  organization        = stackweaver_organization.test-organization.name
  organization_scoped = false
}

// Ensure permissions are assigned second
resource "stackweaver_agent_pool_excluded_workspaces" "excluded" {
  agent_pool_id          = stackweaver_agent_pool.test-agent-pool.id
  excluded_workspace_ids = [stackweaver_workspace.test-workspace.id]
}
```

## Argument Reference

The following arguments are supported:

* `agent_pool_id` - (Required) The ID of the agent pool.
* `excluded_workspace_ids` - (Required) IDs of workspaces to be added as excluded workspaces on the agent pool.


## Import

A resource can be imported; use `<AGENT POOL ID>` as the import ID. For example:

```shell
terraform import stackweaver_agent_pool_excluded_workspaces.foobar apool-rW0KoLSlnuNb5adB
```
