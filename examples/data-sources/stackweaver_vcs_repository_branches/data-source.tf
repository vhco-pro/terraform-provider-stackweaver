data "stackweaver_vcs_repository_branches" "app" {
  vcs_connection_id = "vcs-0123456789abcdef"
  owner             = "my-org"
  repo              = "my-app"
}

output "branch_names" {
  value = [for b in data.stackweaver_vcs_repository_branches.app.branches : b.name]
}
