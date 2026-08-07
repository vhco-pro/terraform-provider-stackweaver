---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_variables"
description: |-
  Get information on a workspace variables.
---

# Data Source: stackweaver_variables

This data source is used to retrieve all variables defined in a specified workspace

## Example Usage

For workspace variables:

```hcl
data "stackweaver_workspace" "test" {
  name         = "my-workspace-name"
  organization = "my-org-name"
}

data "stackweaver_variables" "test" {
  workspace_id = data.stackweaver_workspace.test.id
}
```

For variable set variables:

```hcl
data "stackweaver_variable_set" "test" {
  name         = "my-variable-set-name"
  organization = "my-org-name"
}

data "stackweaver_variables" "test" {
  variable_set_id = data.stackweaver_variable_set.test.id
}
```

## Argument Reference

One of following arguments are required:

* `workspace_id` - ID of the workspace.
* `variable_set_id` - ID of the workspace.

## Attributes Reference

* `variables` - List containing all terraform and environment variables configured on the workspace
* `terraform` - List containing terraform variables configured on the workspace
* `env` - List containing environment variables configured on the workspace

The `variables, terraform and env` blocks contains:

* `id` - The variable Id
* `name` - The variable Key name
* `value` -  The variable value. If the variable is sensitive this value will be empty.
* `category` -  The category of the variable (terraform or environment)
* `sensitive` - If the variable is marked as sensitive or not
* `hcl` - If the variable is marked as HCL or not
