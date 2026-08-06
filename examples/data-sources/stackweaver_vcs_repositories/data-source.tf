data "stackweaver_vcs_repositories" "all" {
  vcs_connection_id = "vcs-0123456789abcdef"
}

output "repository_names" {
  value = [for r in data.stackweaver_vcs_repositories.all.repositories : r.full_name]
}
