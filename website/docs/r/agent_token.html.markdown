---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_agent_token"
description: |-
  Manages agent tokens
---

# stackweaver_agent_token

Each agent pool has its own set of tokens which are not shared across pools.
These tokens allow agents to communicate securely with Stackweaver.

## Example Usage

Basic usage:

```hcl
resource "stackweaver_organization" "test-organization" {
  name  = "my-org-name"
  email = "admin@company.com"
}

resource "stackweaver_agent_pool" "test-agent-pool" {
  name         = "my-agent-pool-name"
  organization = stackweaver_organization.test-organization.id
}

resource "stackweaver_agent_token" "test-agent-token" {
  agent_pool_id = stackweaver_agent_pool.test-agent-pool.id
  description   = "my-agent-token-name"
}
```

## Argument Reference

The following arguments are supported:

* `agent_pool_id` - (Required) ID of the agent pool.
* `description` - (Required) Description of the agent token.

## Attributes Reference

* `id` - The ID of the agent token.
* `description` - The description of agent token.
* `token` - The generated token.
