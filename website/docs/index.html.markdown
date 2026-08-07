---
layout: "stackweaver"
page_title: "Provider: Stackweaver"
description: |-
  The Stackweaver provider manages Stackweaver platform resources - organizations, workspaces, projects, teams, variables, runs, agent pools, run tasks, the private registry, and the full Ansible/AWX surface - with Terraform.
---

# Stackweaver Provider

The Stackweaver provider is used to manage resources on the
[Stackweaver](https://stackweaver.io) platform: organizations, workspaces, projects, teams,
variables, run triggers, agent pools, run tasks, the private module/provider registry, and the
Ansible/AWX surface (playbooks, inventories, credentials, job templates, schedules, notifications).

It is a standalone provider **derived from** `hashicorp/terraform-provider-tfe` (MPL-2.0); see the
repository's `FORK-NOTICE.md`. It is **not** an official HashiCorp product and is not affiliated with
or endorsed by HashiCorp.

## Example Usage

```hcl
terraform {
  required_providers {
    stackweaver = {
      source  = "vhco-pro/stackweaver"
      version = "~> 0.1"
    }
  }
}

provider "stackweaver" {
  hostname     = "app.stackweaver.io" # or your self-hosted host
  token        = var.stackweaver_token
  organization = "my-org"
}

resource "stackweaver_project" "example" {
  organization = "my-org"
  name         = "example"
}
```

## Authentication

The provider authenticates with a Stackweaver API token, supplied via the `token` argument or the
`TFE_TOKEN` environment variable, and targets a host via the `hostname` argument or `TFE_HOSTNAME`
(the `TFE_*` variable names are retained for drop-in compatibility). Tokens can be generated from the
Stackweaver UI.

## `tfe_*` aliases and migrating from terraform-provider-tfe

Every Stackweaver-compatible resource is available under **both** its native `stackweaver_*` name and
a `tfe_*` alias, so existing Terraform Cloud / Enterprise configurations are a drop-in migration:
point the `tfe` provider's `source` at `vhco-pro/stackweaver` and your existing `resource "tfe_*"`
blocks keep working. Once migrated, rename resources to `stackweaver_*` with `moved {}` blocks or
`terraform state mv`.

Stackweaver-native resources that have no `terraform-provider-tfe` equivalent (the Ansible surface,
runner and VCS data sources, and so on) are exposed under `stackweaver_*` only.

## Argument Reference

The provider configuration block accepts:

* `hostname` - (Optional) The Stackweaver hostname. Defaults to `app.stackweaver.io`, or the
  `TFE_HOSTNAME` environment variable.
* `token` - (Optional) A Stackweaver API token. May also be set with the `TFE_TOKEN` environment
  variable or a Terraform CLI credentials block.
* `organization` - (Optional) The default organization for resources that do not set one explicitly.
* `ssl_skip_verify` - (Optional) Skip TLS certificate verification. Defaults to `false`.
