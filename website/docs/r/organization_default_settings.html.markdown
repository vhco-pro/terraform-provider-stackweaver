---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_organization_default_settings
description: |-
  Sets the workspace defaults for an organization
---

# stackweaver_organization_default_settings

Primarily, this is used to set the default execution mode of an organization. Settings configured here will be used as the default for all workspaces in the organization, unless they specify their own values with a [`stackweaver_workspace_settings` resource](workspace_settings.html) (or deprecated attributes on the workspace resource).

## Example Usage

Basic usage:

```hcl
resource "stackweaver_organization" "test" {
  name  = "my-org-name"
  email = "admin@company.com"
}

resource "stackweaver_agent_pool" "my_agents" {
  name         = "agent_smiths"
  organization = stackweaver_organization.test.name
}

resource "stackweaver_organization_default_settings" "org_default" {
  organization           = stackweaver_organization.test.name
  default_execution_mode = "agent"
  default_agent_pool_id  = stackweaver_agent_pool.my_agents.id
}

resource "stackweaver_workspace" "my_workspace" {
  name       = "my-workspace"
  # This workspace will use the org defaults, and will report those defaults as
  # the values of its corresponding attributes. Use depends_on to get accurate
  # values immediately, and to ensure reliable behavior of stackweaver_workspace_run.
  depends_on = [stackweaver_organization_default_settings.org_default]
}
```

## Argument Reference

The following arguments are supported:

* `default_execution_mode` - (Optional) Which [execution mode](https://developer.hashicorp.com/terraform/cloud-docs/workspaces/settings#execution-mode)
  to use as the default for all workspaces in the organization. Valid values are `remote`, `local` or`agent`.
* `default_agent_pool_id` - (Optional) The ID of an agent pool to assign to the workspace. Requires `default_execution_mode` to be set to `agent`. This value _must not_ be provided if `default_execution_mode` is set to any other value.
* `organization` - (Optional) Name of the organization. If omitted, organization must be defined in the provider config.


## Import

Organization default execution mode can be imported; use `<ORGANIZATION NAME>` as the import ID. For example:

```shell
terraform import stackweaver_organization_default_settings.test my-org-name
```
