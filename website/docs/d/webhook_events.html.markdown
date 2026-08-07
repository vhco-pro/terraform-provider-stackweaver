---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_webhook_events"
subcategory: "VCS"
description: |-
  Lists recent VCS webhook deliveries recorded for an organization, newest first.
---

# stackweaver_webhook_events (Data Source)

Use this data source to list recent VCS webhook deliveries recorded for an organization, newest first.
It is an audit/debug helper for inspecting the delivery log. The raw webhook payload is never exposed.

This is a native Stackweaver data source with no `terraform-provider-tfe` equivalent.

## Example Usage

```hcl
data "stackweaver_webhook_events" "recent" {
  organization = "my-org-name"
}

output "failed_deliveries" {
  value = [
    for e in data.stackweaver_webhook_events.recent.events : e.id
    if e.status == "failed"
  ]
}
```

## Argument Reference

The following arguments are supported:

* `organization` - (Optional) Name of the organization. Defaults to the provider's organization.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - Set to the organization name.
* `events` - Recent webhook deliveries, newest first. Each element documented below.

The `events` block contains:

* `id` - Delivery ID.
* `event_type` - Event type: `push`, `pull_request`, `ping`, `installation`, and so on.
* `provider` - VCS provider: `github`, `gitlab`, and so on.
* `repository` - Repository in `owner/repo` format, when applicable.
* `branch` - Branch, when applicable.
* `commit` - Commit SHA, when applicable.
* `status` - Delivery status: `success`, `failed`, or `ignored`.
* `response_code` - HTTP code Stackweaver returned.
* `message` - Additional info / error message.
* `delivered_at` - RFC3339 receipt time.
* `processed_at` - RFC3339 processing time; null until processed.
