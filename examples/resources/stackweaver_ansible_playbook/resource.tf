# A VCS-backed Ansible playbook registered within a project.
# The owning organization is taken from the provider configuration.
resource "stackweaver_ansible_playbook" "example" {
  project_id        = "6f9e2c1a-4b7d-4e5f-8a1b-2c3d4e5f6a7b"
  name              = "site"
  description       = "Primary site playbook"
  vcs_connection_id = "b1c2d3e4-5f6a-7b8c-9d0e-1f2a3b4c5d6e"
  vcs_repository    = "octocat/ansible-playbooks"
  vcs_branch        = "main"
  playbook_path     = "playbooks/site.yml"
}
