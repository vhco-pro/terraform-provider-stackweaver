---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_organizations"
description: |-
  Gets a list of organizations and a map of their IDs.
---

# Data Source: stackweaver_organizations

Gets the organizations the token can see, and a map of their IDs. A user token sees every
organization its user belongs to; an organization-scoped token sees only its own organization. Also
available as `tfe_organizations`; set `provider = stackweaver` on a block that uses the `tfe_` name.

## Example Usage

```terraform
data "stackweaver_organizations" "all" {}
```

## Schema

### Optional

- `admin` (Boolean) Not supported by Stackweaver, which has no site-wide organization listing. Leave it unset; the default listing already returns every organization the token can see.

### Read-Only

- `id` (String) Static identifier for this data source. Do not rely on this value.
- `ids` (Map of String) A map of organization names to their UUIDs.
- `names` (List of String) The names of the organizations.
