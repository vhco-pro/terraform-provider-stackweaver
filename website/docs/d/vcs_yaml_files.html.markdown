---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_vcs_yaml_files"
subcategory: "VCS"
description: |-
  Lists candidate YAML (or inventory) file paths in a repository reachable through a Stackweaver VCS connection.
---

# stackweaver_vcs_yaml_files (Data Source)

Use this data source to list candidate YAML file paths - or, in inventory mode, inventory file paths -
inside a repository reachable through a Stackweaver VCS connection. A single data source covers both file
sets via `file_type`, since they differ only in the server-side extension filter.

This is a native Stackweaver data source with no `terraform-provider-tfe` equivalent.

## Example Usage

```hcl
data "stackweaver_vcs_yaml_files" "playbooks" {
  vcs_connection_id = "vcs-0123456789abcdef"
  owner             = "my-org"
  repo              = "my-ansible"
  file_type         = "playbook"
  ref               = "main"
}

output "playbook_paths" {
  value = data.stackweaver_vcs_yaml_files.playbooks.paths
}
```

## Argument Reference

The following arguments are supported:

* `vcs_connection_id` - (Required) ID of the VCS connection.
* `owner` - (Required) Repository owner (org/user/project).
* `repo` - (Required) Repository name.
* `ref` - (Optional) Branch or commit to read. Empty reads the default branch.
* `file_type` - (Optional) Which file set to list: `playbook` (default → `.yaml`/`.yml`) or `inventory`
  (→ `.ini`/`.yaml`/`.yml`/`.json`).

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - Synthetic identifier - `<connection_id>/<owner>/<repo>/<file_type>@<ref>`.
* `paths` - Repo-relative paths of the matching files.
