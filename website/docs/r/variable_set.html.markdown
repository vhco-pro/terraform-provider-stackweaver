---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_variable_set"
description: |-
  Manages variable sets.
---

# stackweaver_variable_set

Creates, updates and destroys variable sets.

## Example Usage

Basic usage:

```hcl
resource "stackweaver_organization" "test" {
  name  = "my-org-name"
  email = "admin@company.com"
}

resource "stackweaver_project" "test" {
  organization = stackweaver_organization.test.name
  name = "projectname"
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
  workspace_id    = stackweaver_workspace.test.id
  variable_set_id = stackweaver_variable_set.test.id
}

resource "stackweaver_project_variable_set" "test" {
  project_id    = stackweaver_project.test.id
  variable_set_id = stackweaver_variable_set.test.id
}

resource "stackweaver_stack_variable_set" "test" {
  stack_id        = stackweaver_stack.test.id
  variable_set_id = stackweaver_variable_set.test.id
}

resource "stackweaver_variable" "test-a" {
  key             = "seperate_variable"
  value           = "my_value_name"
  category        = "terraform"
  description     = "a useful description"
  variable_set_id = stackweaver_variable_set.test.id
}

resource "stackweaver_variable" "test-b" {
  key             = "another_variable"
  value           = "my_value_name"
  category        = "env"
  description     = "an environment variable"
  variable_set_id = stackweaver_variable_set.test.id
}
```

Creating a global variable set:

```hcl
resource "stackweaver_organization" "test" {
  name  = "my-org-name"
  email = "admin@company.com"
}

resource "stackweaver_variable_set" "test" {
  name         = "Global Varset"
  description  = "Variable set applied to all workspaces."
  global       = true
  organization = stackweaver_organization.test.name
}

resource "stackweaver_variable" "test-a" {
  key             = "seperate_variable"
  value           = "my_value_name"
  category        = "terraform"
  description     = "a useful description"
  variable_set_id = stackweaver_variable_set.test.id
}

resource "stackweaver_variable" "test-b" {
  key             = "another_variable"
  value           = "my_value_name"
  category        = "env"
  description     = "an environment variable"
  variable_set_id = stackweaver_variable_set.test.id
}
```

Create a priority variable set:

```hcl
resource "stackweaver_organization" "test" {
  name  = "my-org-name"
  email = "admin@company.com"
}

resource "stackweaver_variable_set" "test" {
  name         = "Global Varset"
  description  = "Variable set applied to all workspaces."
  priority     = true
  organization = stackweaver_organization.test.name
}

resource "stackweaver_variable" "test-a" {
  key             = "seperate_variable"
  value           = "my_value_name"
  category        = "terraform"
  description     = "a useful description"
  variable_set_id = stackweaver_variable_set.test.id
}

resource "stackweaver_variable" "test-b" {
  key             = "another_variable"
  value           = "my_value_name"
  category        = "env"
  description     = "an environment variable"
  variable_set_id = stackweaver_variable_set.test.id
}
```

Creating a project-owned variable set that is applied to all workspaces in the project:

```hcl
resource "stackweaver_organization" "test" {
  name  = "my-org-name"
  email = "admin@company.com"
}

resource "stackweaver_project" "test" {
  organization = stackweaver_organization.test.name
  name = "projectname"
}

resource "stackweaver_variable_set" "test" {
  name              = "Project-owned Varset"
  description       = "Varset that is owned and managed by a project."
  organization      = stackweaver_organization.test.name
  parent_project_id = stackweaver_project.test.id
}

resource "stackweaver_project_variable_set" "test" {
  project_id      = stackweaver_project.test.id
  variable_set_id = stackweaver_variable_set.test.id
}
```

Creating a project-owned variable set that is applied to specific workspaces:

```hcl
resource "stackweaver_organization" "test" {
  name  = "my-org-name"
  email = "admin@company.com"
}

resource "stackweaver_project" "test" {
  organization = stackweaver_organization.test.name
  name = "projectname"
}

resource "stackweaver_workspace" "test" {
  name         = "my-workspace-name"
  organization = stackweaver_organization.test.name
  project_id   = stackweaver_project.test.id
}

resource "stackweaver_variable_set" "test" {
  name              = "Project-owned Varset"
  description       = "Varset that is owned and managed by a project."
  organization      = stackweaver_organization.test.name
  parent_project_id = stackweaver_project.test.id
}

resource "stackweaver_workspace_variable_set" "test" {
  workspace_id    = stackweaver_workspace.test.id
  variable_set_id = stackweaver_variable_set.test.id
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Name of the variable set.
* `description` - (Optional) Description of the variable set.
* `global` - (Optional) Whether the variable set applies to all workspaces in the organization. Defaults to `false`.
* `priority` - (Optional) Whether the variables in this set can be over-written by more specific scopes including values set on the command line. Defaults to `false`.
* `organization` - (Optional) Name of the organization. If omitted, organization must be defined in the provider config.
* `workspace_ids` - **Deprecated** (Optional) IDs of the workspaces that use the variable set.
  Must not be set if `global` is set. This argument is mutually exclusive with using the resource
  [stackweaver_workspace_variable_set](workspace_variable_set.html) which is the preferred method of associating a workspace
  with a variable set.
* `parent_project_id` - (Optional) ID of the project that should own the variable set. If set, than the value of `global` must be `false`.
  To assign whether a variable set should be applied to a project, use the [`stackweaver_project_variable_set`](project_variable_set.html) resource.

## Attributes Reference

* `id` - The ID of the variable set.

## Import

Variable sets can be imported using an identity. For example:

```hcl
import {
  to = stackweaver_variable_set.test
  identity = {
    id       = "varset-kjkN545LH2Sfercv"
    hostname = "app.terraform.io"
  }
}
```

Variable sets can be imported using the Terraform CLI; use `<VARIABLE SET ID>` as the import ID. For example:

```shell
terraform import stackweaver_variable_set.test varset-5rTwnSaRPogw6apb
```
