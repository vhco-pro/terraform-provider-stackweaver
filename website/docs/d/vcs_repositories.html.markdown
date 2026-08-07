---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_vcs_repositories"
subcategory: "VCS"
description: |-
  Lists the repositories reachable through a Stackweaver VCS connection.
---

# stackweaver_vcs_repositories (Data Source)

Use this data source to list the repositories reachable through a Stackweaver VCS connection. It is a
read-only discovery helper - useful for populating workspace or playbook configuration from the set of
repositories a connection can see.

This is a native Stackweaver data source with no `terraform-provider-tfe` equivalent.

## Example Usage

```hcl
data "stackweaver_vcs_repositories" "all" {
  vcs_connection_id = "vcs-0123456789abcdef"
}

output "repository_names" {
  value = [for r in data.stackweaver_vcs_repositories.all.repositories : r.full_name]
}
```

## Argument Reference

The following arguments are supported:

* `vcs_connection_id` - (Required) ID of the VCS connection to enumerate.
* `project` - (Optional) Azure DevOps project scope. Ignored by providers without a project layer.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - Synthetic identifier - the VCS connection ID.
* `repositories` - The repositories reachable through the connection. Each element documented below.

The `repositories` block contains:

* `id` - Provider numeric repository ID.
* `name` - Short repository name.
* `full_name` - Full repository name (`owner/repo`).
* `description` - Repository description.
* `private` - Whether the repository is private.
* `default_branch` - Default branch name.
* `url` - Web URL of the repository.
* `clone_url` - HTTPS clone URL.
* `ssh_url` - SSH clone URL.
