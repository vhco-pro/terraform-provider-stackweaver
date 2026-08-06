# A group attached to a static inventory.
resource "stackweaver_ansible_inventory" "example" {
  organization = "my-org"
  name         = "production"
  type         = "static"
}

resource "stackweaver_ansible_group" "webservers" {
  inventory_id = stackweaver_ansible_inventory.example.id
  name         = "webservers"
  description  = "Front-end web tier"
  variables = {
    http_port = "8080"
  }
}
