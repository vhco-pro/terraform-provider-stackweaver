variable "aws_access_key_id" {
  type      = string
  sensitive = true
}

variable "aws_secret_access_key" {
  type      = string
  sensitive = true
}

resource "stackweaver_ansible_credential" "aws" {
  organization          = "my-org-name"
  name                  = "aws-inventory"
  credential_type       = "aws"
  aws_access_key_id     = var.aws_access_key_id
  aws_secret_access_key = var.aws_secret_access_key
}

resource "stackweaver_ansible_inventory_source" "aws" {
  inventory_id  = "c1d2e3f4-5a6b-4c7d-8e9f-0a1b2c3d4e5f"
  name          = "aws-ec2"
  source_type   = "aws"
  credential_id = stackweaver_ansible_credential.aws.id

  config = jsonencode({
    regions = ["eu-west-1"]
  })

  update_on_launch = true
}
