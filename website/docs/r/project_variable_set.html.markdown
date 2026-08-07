---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_project_variable_set"
description: |-
  Add a variable set to a project
---

# stackweaver_project_variable_set

Adds and removes a project from a variable set's scope.

-> **Note:** This resource controls whether a project has access to a variable set, not whether
a project owns the variable set. Ownership is specified by setting the `parent_project_id` on the
`stackweaver_variable_set` resource.

## Example Usage

Basic usage:

```hcl
resource "stackweaver_organization" "test" {
  name  = "my-org-name"
  email = "admin@company.com"
}

resource "stackweaver_project" "test" {
  name         = "my-project-name"
  organization = stackweaver_organization.test.name
}

resource "stackweaver_variable_set" "test" {
  name         = "Test Varset"
  description  = "Some description."
  organization = stackweaver_organization.test.name
}

resource "stackweaver_project_variable_set" "test" {
  variable_set_id = stackweaver_variable_set.test.id
  project_id      = stackweaver_project.test.id
}
```

## Argument Reference

The following arguments are supported:

* `variable_set_id` - (Required) Name of the variable set to add.
* `project_id` - (Required) Project ID to add the variable set to.

## Attributes Reference

* `id` - The ID of the variable set attachment. ID format: `<project-id>_<variable-set-id>`

## Import

Project Variable Sets can be imported; use `<ORGANIZATION>/<PROJECT ID>/<VARIABLE SET NAME>`. For example:

```shell
terraform import stackweaver_project_variable_set.test 'my-org-name/prj-F1NpdVBuCF3xc5Rp/Test Varset'
```
