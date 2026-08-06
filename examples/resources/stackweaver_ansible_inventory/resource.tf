# A static, organization-scoped Ansible inventory.
resource "stackweaver_ansible_inventory" "example" {
  organization = "my-org"
  name         = "production"
  type         = "static"
}
