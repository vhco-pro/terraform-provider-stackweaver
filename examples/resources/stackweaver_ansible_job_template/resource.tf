resource "stackweaver_ansible_job_template" "deploy" {
  organization = "my-org-name"
  name         = "deploy-web"
  description  = "Deploy the web tier"

  playbook_id  = stackweaver_ansible_playbook.site.id
  inventory_id = stackweaver_ansible_inventory.production.id

  limit           = "web"
  tags            = "deploy"
  verbosity       = 1
  forks           = 10
  become_enabled  = true
  diff_mode       = true
  timeout_seconds = 3600

  extra_vars = {
    app_version = "1.2.3"
  }
}

# Credentials are attached as their own resources: the template holds a set of
# them (at most one per type, plus any number of vault credentials with distinct
# vault IDs) rather than a single credential field.
resource "stackweaver_ansible_job_template_credential" "deploy_ssh" {
  job_template_id = stackweaver_ansible_job_template.deploy.id
  credential_id   = stackweaver_ansible_credential.ssh.id
}
