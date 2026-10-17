# Creating an organization needs a user token; an organization-scoped token is refused.
resource "stackweaver_organization" "example" {
  name  = "my-org-name"
  email = "admin@example.com"
}
