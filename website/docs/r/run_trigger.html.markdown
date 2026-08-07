---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_run_trigger"
description: |-
  Manages run triggers
---

# stackweaver_run_trigger

Stackweaver provides a way to connect your workspace to one or more workspaces within your organization, 
known as "source workspaces". These connections, called run triggers, allow runs to queue automatically in 
your workspace on successful apply of runs in any of the source workspaces. You can connect your workspace 
to up to 20 source workspaces.

## Example Usage

Basic usage:

```hcl
resource "stackweaver_organization" "test-organization" {
  name  = "my-org-name"
  email = "admin@company.com"
}

resource "stackweaver_workspace" "test-workspace" {
  name         = "my-workspace-name"
  organization = stackweaver_organization.test-organization.id
}

resource "stackweaver_workspace" "test-sourceable" {
  name         = "my-sourceable-workspace-name"
  organization = stackweaver_organization.test-organization.id
}

resource "stackweaver_run_trigger" "test" {
  workspace_id  = stackweaver_workspace.test-workspace.id
  sourceable_id = stackweaver_workspace.test-sourceable.id
}
```

## Argument Reference

The following arguments are supported:

* `workspace_id` - (Required) The id of the workspace that owns the run trigger. This is the 
  workspace where runs will be triggered.
* `sourceable_id` - (Required) The id of the sourceable. The sourceable must be a workspace.

## Attributes Reference

* `id` - The ID of the run trigger.

## Import

Run triggers can be imported; use `<RUN TRIGGER ID>` as the import ID. For example:

```shell
terraform import stackweaver_run_trigger.test rt-qV9JnKRkmtMa4zcA
```
