---
layout: "stackweaver"
subcategory: "Ansible"
page_title: "Stackweaver: stackweaver_ansible_playbook"
description: |-
  Registers an Ansible playbook backed by a VCS repository.
---

# stackweaver_ansible_playbook

Registers an Ansible playbook: a named pointer, within a project, at a playbook
file in a VCS repository. Job templates and jobs reference the playbook, and a
sync action pulls the repository at the pinned branch and path.

This is a native Stackweaver resource with no Terraform Enterprise equivalent.

## Example Usage

Basic usage:

```hcl
resource "stackweaver_ansible_playbook" "example" {
  project_id        = "6f9e2c1a-4b7d-4e5f-8a1b-2c3d4e5f6a7b"
  name              = "site"
  description       = "Primary site playbook"
  vcs_connection_id = "b1c2d3e4-5f6a-7b8c-9d0e-1f2a3b4c5d6e"
  vcs_repository    = "octocat/ansible-playbooks"
  vcs_branch        = "main"
  playbook_path     = "playbooks/site.yml"
}
```

## Argument Reference

The following arguments are supported:

* `project_id` - (Required) ID of the owning project. The pair `(project_id, name)`
  is unique. Changing this forces a new playbook to be created.
* `name` - (Required) Name of the playbook, unique within the project.
* `description` - (Optional) Human-readable description of the playbook.
* `vcs_connection_id` - (Optional) ID of the VCS connection to pull the playbook
  repository from.
* `vcs_repository` - (Optional) Full name of the repository, in `"owner/repo"`
  form.
* `vcs_branch` - (Optional) Branch to pull. Defaults to `main`.
* `playbook_path` - (Optional) Path to the playbook file within the repository.
  Defaults to `site.yml`.
* `source_mode` - (Optional) Where a job sources the playbook content from:
  `cached` (default) or `fresh`.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The playbook ID.
* `last_sync_at` - Timestamp of the most recent VCS sync.
* `last_sync_status` - Status of the most recent VCS sync.
* `last_sync_commit` - Git commit SHA of the most recent VCS sync.
* `cached_commit` - Git commit SHA of the cached playbook snapshot.
* `cached_at` - Timestamp of the cached playbook snapshot.

## Import

Ansible playbooks can be imported using their ID. For example:

```shell
terraform import stackweaver_ansible_playbook.example 6f9e2c1a-4b7d-4e5f-8a1b-2c3d4e5f6a7b
```
