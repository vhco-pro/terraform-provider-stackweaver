# Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0
#
# Run with the main fixture's organization-scoped token.
#   - Resource criterion 7: creating an organization fails with 403 and creates nothing.
#   - Organizations data source criterion 2: the listing is exactly the token's organization.
# ./run.sh applies `-target=data.stackweaver_organizations.scoped` first, then the full config
# expecting the 403.

terraform {
  required_providers {
    stackweaver = {
      source = "vhco-pro/stackweaver"
    }
  }
}

variable "run_id" {
  type = string
}

provider "stackweaver" {}

data "stackweaver_organizations" "scoped" {}

resource "stackweaver_organization" "refused" {
  name  = "pv-refused-${var.run_id}"
  email = "provider-acceptance@example.test"
}

output "scoped_names" {
  value = data.stackweaver_organizations.scoped.names
}
