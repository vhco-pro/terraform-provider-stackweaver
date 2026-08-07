---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_workspace_variable_set"
description: |-
  Add a variable set to a workspace
---

# stackweaver_workspace_variable_set

Adds and removes a workspace from a variable set's scope.

-> **Note:** `stackweaver_variable_set` has a deprecated argument `workspace_ids` that should not be used alongside this resource. They attempt to manage the same attachments and are mutually exclusive.

## Example Usage

Basic usage:

```hcl
resource "stackweaver_organization" "test" {
  name  = "my-org-name"
  email = "admin@company.com"
}

resource "stackweaver_workspace" "test" {
  name         = "my-workspace-name"
  organization = stackweaver_organization.test.name
}

resource "stackweaver_variable_set" "test" {
  name          = "Test Varset"
  description   = "Some description."
  organization  = stackweaver_organization.test.name
}

resource "stackweaver_workspace_variable_set" "test" {
  variable_set_id = stackweaver_variable_set.test.id
  workspace_id    = stackweaver_workspace.test.id
}
```

## Argument Reference

The following arguments are supported:

* `variable_set_id` - (Required) The variable set ID.
* `workspace_id` - (Required) Workspace ID to add the variable set to.

## Attributes Reference

* `id` - The ID of the variable set attachment. ID format: `<workspace-id>_<variable-set-id>`

## Import

Workspace Variable Sets can be imported; use `<ORGANIZATION>/<WORKSPACE NAME>/<VARIABLE SET NAME>`. For example:

```shell
terraform import stackweaver_workspace_variable_set.test 'my-org-name/workspace/My Variable Set'
```
