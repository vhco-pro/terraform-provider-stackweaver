resource "stackweaver_ansible_job_template_credential" "vault" {
  job_template_id = stackweaver_ansible_job_template.deploy.id
  credential_id   = stackweaver_ansible_credential.vault.id
}
