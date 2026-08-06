data "stackweaver_runners" "terraform" {
  organization = "my-org-name"
  runner_type  = "terraform"
}

output "online_runners" {
  value = data.stackweaver_runners.terraform.stats.online
}
