# A host attached to a static inventory.
resource "stackweaver_ansible_inventory" "example" {
  organization = "my-org"
  name         = "production"
  type         = "static"
}

resource "stackweaver_ansible_host" "web" {
  inventory_id = stackweaver_ansible_inventory.example.id
  name         = "web-01"
  hostname     = "10.0.0.10"
  port         = 22
  variables = {
    ansible_user = "deploy"
  }
}
