---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_ansible_config"
subcategory: "Ansible"
description: |-
  Manages the ansible.cfg content for a scope.
---

# stackweaver_ansible_config

Manages the raw `ansible.cfg` content for a scope. This is a native Stackweaver resource with no
`terraform-provider-tfe` equivalent.

There is exactly **one config per scope entity** (a singleton): a single row keyed by exactly one of
organization or project. At run time the runner renders `ansible.cfg` from the most-specific config
for the run's scope, with resolution priority Workspace > Project > Organization, so this content
directly changes Ansible behavior (host key checking, callback plugins, forks, and so on).

The config is upserted via PUT: the first apply creates the row and subsequent applies update it in
place. Exactly one of `organization` or `project_id` must be set, and the scope selector is immutable
- changing it targets a different singleton and forces a new resource.

Workspace scope is not manageable through this resource. The API echoes `workspace_id` in its
response, but exposes no workspace route to set it, so `workspace_id` and `scope` are read-only here
and the resource covers organization and project scopes only.

## Example Usage

A project-scoped `ansible.cfg` using a heredoc:

```hcl
resource "stackweaver_ansible_config" "project" {
  project_id = "b7d2f1a4-3c8e-4d6b-9a1f-5e2c8d4b7a3f"

  config_content = <<-EOT
    [defaults]
    host_key_checking = False
    forks             = 25
    stdout_callback   = yaml
  EOT
}
```

An organization-scoped config:

```hcl
resource "stackweaver_ansible_config" "org" {
  organization = "my-org-name"

  config_content = <<-EOT
    [defaults]
    host_key_checking = False
  EOT
}
```

## Argument Reference

The following arguments are supported:

* `config_content` - (Required) The raw `ansible.cfg` content. Updated in place via the same PUT
  upsert.
* `organization` - (Optional) Name of the organization for an org-scoped config. Exactly one of
  `organization` or `project_id` must be set. Changing it forces a new config.
* `project_id` - (Optional) ID of the project for a project-scoped config. Exactly one of
  `organization` or `project_id` must be set. Changing it forces a new config.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The config ID.
* `scope` - Server-returned scope of the config: `organization`, `project`, or `workspace`.
* `workspace_id` - Workspace ID when the config is workspace-scoped. Computed only - there is no
  workspace route to set it via this resource.
* `created_at` - Timestamp when the config was created.
* `updated_at` - Timestamp when the config was last updated.

## Import

Ansible configs can be imported using their ID. For example:

```shell
terraform import stackweaver_ansible_config.project 1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d
```
