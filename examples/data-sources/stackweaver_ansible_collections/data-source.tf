data "stackweaver_ansible_collections" "installed" {}

output "collection_names" {
  value = [for c in data.stackweaver_ansible_collections.installed.collections : c.name]
}
