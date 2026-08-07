---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_projects"
description: |-
  Get information on projects in an organization.
---

# Data Source: stackweaver_projects

Use this data source to get information about all projects in an organization.

## Example Usage

```hcl
data "stackweaver_projects" "all" {
  organization = "my-org-name"
}
```

## Argument Reference

The following arguments are supported:

* `organization` - (Optional) Name of the organization. If omitted, organization must be defined in the provider config.

## Attributes Reference

* `projects` - List of projects in the organization. Each element contains the following attributes:
  * `id` - ID of the project.
  * `name` - Name of the project.
  * `description` - Description of the organization.
  * `organization` - Name of the organization.
  * `auto_destroy_activity_duration` - A duration string for all workspaces in the project, representing time after each workspace's activity when an auto-destroy run will be triggered.
