resource "stackweaver_ansible_job_template_variable" "app_version" {
  job_template_id = stackweaver_ansible_job_template.deploy.id
  key             = "app_version"
  value           = "1.2.3"
  category        = "env"
  description     = "Version of the application to deploy"
}

resource "stackweaver_ansible_job_template_variable" "api_token" {
  job_template_id = stackweaver_ansible_job_template.deploy.id
  key             = "api_token"
  value           = var.api_token
  sensitive       = true
}
