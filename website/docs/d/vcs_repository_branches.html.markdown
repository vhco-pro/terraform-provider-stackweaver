---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_vcs_repository_branches"
subcategory: "VCS"
description: |-
  Lists the branches of a repository reachable through a Stackweaver VCS connection.
---

# stackweaver_vcs_repository_branches (Data Source)

Use this data source to list the branches of a single repository reachable through a Stackweaver VCS
connection. It is a read-only discovery helper for wiring branch selection into other configuration.

This is a native Stackweaver data source with no `terraform-provider-tfe` equivalent.

## Example Usage

```hcl
data "stackweaver_vcs_repository_branches" "app" {
  vcs_connection_id = "vcs-0123456789abcdef"
  owner             = "my-org"
  repo              = "my-app"
}

output "branch_names" {
  value = [for b in data.stackweaver_vcs_repository_branches.app.branches : b.name]
}
```

## Argument Reference

The following arguments are supported:

* `vcs_connection_id` - (Required) ID of the VCS connection.
* `owner` - (Required) Repository owner (org/user/project — the left half of `owner/repo`).
* `repo` - (Required) Repository name (the right half of `owner/repo`).

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - Synthetic identifier — `<connection_id>/<owner>/<repo>`.
* `branches` - The branches of the repository. Each element documented below.

The `branches` block contains:

* `name` - Branch name.
* `commit_sha` - Head commit SHA (flattened from `commit.sha`).
* `protected` - Whether the branch is protected.
