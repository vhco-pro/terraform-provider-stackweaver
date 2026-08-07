---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_ansible_vcs_playbook_files"
subcategory: "Ansible"
description: |-
  Lists playbook candidate files in a connected VCS repository at a branch, annotated with registration state.
---

# stackweaver_ansible_vcs_playbook_files (Data Source)

Use this data source to list playbook candidate files in a connected VCS repository at a branch, each
annotated with whether it is already registered as a `stackweaver_ansible_playbook`. It is a discovery
helper for onboarding playbooks from a repository.

This is a native Stackweaver data source with no `terraform-provider-tfe` equivalent.

## Example Usage

```hcl
data "stackweaver_ansible_vcs_playbook_files" "candidates" {
  organization      = "my-org-name"
  vcs_connection_id = "vcs-0123456789abcdef"
  repository        = "my-org/my-ansible"
  branch            = "main"
}

output "unregistered_playbooks" {
  value = [
    for f in data.stackweaver_ansible_vcs_playbook_files.candidates.files : f.path
    if !f.registered
  ]
}
```

## Argument Reference

The following arguments are supported:

* `organization` - (Optional) Name of the organization. Defaults to the provider's organization.
* `vcs_connection_id` - (Required) ID of the VCS connection to browse. Must belong to the organization.
* `repository` - (Required) Repository in `owner/repo` format.
* `branch` - (Required) Branch to list files from. The listing and annotation are branch-scoped.
* `path` - (Optional) Repo path prefix; when set, only files under it are returned.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - Synthesized identifier - `organization/connection/repository/branch`.
* `files` - Discovered playbook candidate files. Each element documented below.

The `files` block contains:

* `path` - Repo-relative file path.
* `name` - Base filename.
* `registered` - Whether a playbook is already registered for this path.
* `playbook_id` - ID of the registered playbook (present only when registered).
* `playbook_name` - Name of the registered playbook (present only when registered).
