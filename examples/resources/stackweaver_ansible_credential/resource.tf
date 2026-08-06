variable "ssh_private_key" {
  type      = string
  sensitive = true
}

resource "stackweaver_ansible_credential" "web_ssh" {
  organization    = "my-org-name"
  name            = "web-ssh"
  credential_type = "ssh"
  username        = "ansible"
  ssh_private_key = var.ssh_private_key
}

# An AWS variant for dynamic-inventory discovery:
#
# variable "aws_access_key_id" {
#   type      = string
#   sensitive = true
# }
#
# variable "aws_secret_access_key" {
#   type      = string
#   sensitive = true
# }
#
# resource "stackweaver_ansible_credential" "aws" {
#   organization          = "my-org-name"
#   name                  = "aws-inventory"
#   credential_type       = "aws"
#   aws_access_key_id     = var.aws_access_key_id
#   aws_secret_access_key = var.aws_secret_access_key
# }
