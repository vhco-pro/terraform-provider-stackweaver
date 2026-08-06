provider "stackweaver" {
  hostname     = "stackweaver.example.com"
  token        = var.stackweaver_token
  organization = "my-org"
}

resource "stackweaver_ansible_notification_template" "webhook" {
  name = "ops-webhook"
  type = "webhook"

  config = jsonencode({
    url    = "https://hooks.example.com/ansible"
    method = "POST"
  })
}

# Attach the template to a job template so the channel fires on success.
# Exactly one of job_template_id or workflow_id may be set. Every attribute is
# ForceNew (create/delete only).
resource "stackweaver_ansible_notification_attachment" "deploy_success" {
  notification_template_id = stackweaver_ansible_notification_template.webhook.id
  job_template_id          = stackweaver_ansible_job_template.deploy.id

  on_started = false
  on_success = true
  on_failure = false
}
