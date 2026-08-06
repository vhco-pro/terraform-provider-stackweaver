data "stackweaver_vcs_yaml_files" "playbooks" {
  vcs_connection_id = "vcs-0123456789abcdef"
  owner             = "my-org"
  repo              = "my-ansible"
  file_type         = "playbook"
  ref               = "main"
}

output "playbook_paths" {
  value = data.stackweaver_vcs_yaml_files.playbooks.paths
}
