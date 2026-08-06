resource "stackweaver_ansible_config" "project" {
  project_id = "b7d2f1a4-3c8e-4d6b-9a1f-5e2c8d4b7a3f"

  config_content = <<-EOT
    [defaults]
    host_key_checking = False
    forks             = 25
    stdout_callback   = yaml
  EOT
}
