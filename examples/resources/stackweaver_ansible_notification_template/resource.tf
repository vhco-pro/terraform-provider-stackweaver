provider "stackweaver" {
  hostname     = "stackweaver.example.com"
  token        = var.stackweaver_token
  organization = "my-org"
}

# A webhook notification template. The organization is taken from the provider
# configuration; it is not an argument on this resource.
resource "stackweaver_ansible_notification_template" "webhook" {
  name = "ops-webhook"
  type = "webhook"

  config = jsonencode({
    url    = "https://hooks.example.com/ansible"
    method = "POST"
    headers = {
      "X-Source" = "stackweaver"
    }
    username        = "ci"
    skip_tls_verify = false
  })

  # Write-only: sent on create/update, never read back from the API.
  secret = var.webhook_password
}
