---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_team_organization_members"
description: |-
  Add or remove users from a team based on their organization memberships.
---

# stackweaver_team_organization_members

Add or remove one or more team members using a
[stackweaver_organization_membership](organization_membership.html).

~> **NOTE** on managing team memberships: Terraform currently provides four
resources for managing team memberships. This - along with [stackweaver_team_organization_member](team_organization_member.html) - is the preferred method as it
allows you to add members to a team by email addresses. The [stackweaver_team_organization_member](team_organization_member.html) is used to manage a single team membership whereas [stackweaver_team_organization_members](team_organization_members.html) is used to manage all team memberships at once. All four resources cannot be used for the same team simultaneously.

~> **NOTE:** This resource requires using the provider with Stackweaver or
an instance of Stackweaver at least as recent as v202004-1.

## Example Usage

Basic usage:

```hcl
resource "stackweaver_team" "test" {
  name         = "my-team-name"
  organization = "my-org-name"
}

resource "stackweaver_organization_membership" "test" {
  organization = "my-org-name"
  email = "admin@example.com"
}

resource "stackweaver_organization_membership" "sample" {
  organization = "my-org-name"
  email = "sample@example.com"
}

resource "stackweaver_team_organization_members" "test" {
  team_id = stackweaver_team.test.id
  organization_membership_ids = [
    stackweaver_organization_membership.test.id,
    stackweaver_organization_membership.sample.id
  ]
}
```

With a set of organization members:

```hcl
locals {
  all_users = toset([
    "user1@example.com",
    "user2@example.com",
  ])
}

resource "stackweaver_team" "test" {
  name         = "my-team-name"
  organization = "my-org-name"
}

resource "stackweaver_organization_membership" "all_membership" {
  organization = "my-org-name"
  for_each     = local.all_users
  email        = each.key
}

resource "stackweaver_team_organization_members" "test" {
  team_id = stackweaver_team.test.id
  organization_membership_ids = [for member in local.all_users : stackweaver_organization_membership.all_membership[member].id]
}
```

## Argument Reference

The following arguments are supported:

* `team_id` - (Required) ID of the team.
* `organization_membership_ids` - (Required) IDs of organization memberships to be added.

## Import

A resource can be imported by using the team ID `<TEAM ID>`
as the import ID. For example:

```shell
terraform import stackweaver_team_organization_members.test team-47qC3LmA47piVan7
```
