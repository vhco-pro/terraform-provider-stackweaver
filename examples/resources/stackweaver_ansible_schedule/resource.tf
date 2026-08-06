provider "stackweaver" {
  hostname     = "stackweaver.example.com"
  token        = var.stackweaver_token
  organization = "my-org"
}

# A daily job-template schedule. The target id must match `type`; exactly one of
# job_template_id / inventory_source_id / playbook_id / workflow_id may be set.
resource "stackweaver_ansible_schedule" "nightly" {
  name            = "nightly-deploy"
  type            = "job_template"
  job_template_id = stackweaver_ansible_job_template.deploy.id

  cron_expression = "0 2 * * *"
  timezone        = "America/New_York"

  config = jsonencode({
    extra_vars = {
      environment = "staging"
    }
  })
}
