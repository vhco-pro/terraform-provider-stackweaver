<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
# Terraform Provider for Stackweaver

The official Terraform provider for the [Stackweaver](https://stackweaver.io) platform - manage
workspaces, projects, teams, variables, runs, agent pools, run tasks, the registry, and the full
Ansible/AWX surface (playbooks, inventories, credentials, job templates, schedules, notifications) as
code.

It is a standalone provider **derived from** [`hashicorp/terraform-provider-tfe`](https://github.com/hashicorp/terraform-provider-tfe)
(MPL-2.0); see [`FORK-NOTICE.md`](./FORK-NOTICE.md). It is **not** an official HashiCorp product and is
not affiliated with or endorsed by HashiCorp.

## Resource naming

Every TFE-compatible resource is available under a native `stackweaver_*` name **and** a `tfe_*`
alias, so existing Terraform Cloud / Enterprise configurations are a drop-in migration:

```hcl
terraform {
  required_providers {
    stackweaver = {
      source = "vhco-pro/stackweaver"
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

Migrating from `terraform-provider-tfe`? Point the `tfe` provider's `source` at
`vhco-pro/stackweaver` - your `resource "tfe_*"` blocks keep working - then optionally rename to
`stackweaver_*` with `moved {}` blocks.

Stackweaver-native resources (no TFE equivalent, e.g. the Ansible surface) are exposed under
`stackweaver_*` only.

## Documentation

Per-resource docs and examples live under [`website/docs/`](./website/docs/) and
[`examples/`](./examples/), and are published to the Terraform Registry.

## Development

```sh
make build        # build the provider
make test         # unit tests (native codecs + registration)
make fmt vet lint # format, vet, lint
```

Run acceptance tests against a live Stackweaver stack with a Terraform CLI dev override
(`~/.terraformrc` `dev_overrides` → the built binary; see the Registry docs). CI runs build + unit;
full acceptance is driven locally because it needs a running stack.

## License

[Mozilla Public License 2.0](./LICENSE). Upstream `Copyright (c) HashiCorp, Inc.` headers are
preserved on forked files.
