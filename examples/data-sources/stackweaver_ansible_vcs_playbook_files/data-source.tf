data "stackweaver_ansible_vcs_playbook_files" "candidates" {
  organization      = "my-org-name"
  vcs_connection_id = "vcs-0123456789abcdef"
  repository        = "my-org/my-ansible"
  branch            = "main"
}

output "unregistered_playbooks" {
  value = [
    for f in data.stackweaver_ansible_vcs_playbook_files.candidates.files : f.path
    if !f.registered
  ]
}
