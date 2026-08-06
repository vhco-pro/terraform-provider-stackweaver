data "stackweaver_ansible_adhoc_modules" "allowed" {
  organization = "my-org-name"
}

output "adhoc_modules" {
  value = data.stackweaver_ansible_adhoc_modules.allowed.modules
}
