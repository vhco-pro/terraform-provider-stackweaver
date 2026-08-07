---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_aws_oidc_configuration"
description: |-
  Manages AWS OIDC configurations.
---

# stackweaver_aws_oidc_configuration

Defines an AWS OIDC configuration resource.

~> **NOTE:** This resource requires using the provider with Stackweaver on the Stackweaver Premium edition. Refer to [Stackweaver pricing](https://stackweaver.io/pricing) for details.

## Example Usage

Basic usage:

```hcl
resource "stackweaver_aws_oidc_configuration" "example" {
  role_arn      = "arn:aws:iam::111111111111:role/example-role-arn"
  organization  = "my-org-name"
}
```


## Argument Reference

The following arguments are supported:

* `role_arn` - (Required) The AWS ARN of your role..
* `organization` - (Optional) Name of the organization. If omitted, organization must be defined in the provider config.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The AWS OIDC configuration ID.

## Import
AWS OIDC configurations can be imported by ID.

Example:

```shell
terraform import stackweaver_aws_oidc_configuration.example awsoidc-DXmy3B2emVHysnbq
```
