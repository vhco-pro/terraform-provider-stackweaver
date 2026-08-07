provider "stackweaver" {
  hostname     = "stackweaver.example.com"
  token        = var.stackweaver_token
  organization = "my-org"
}

# Launch a job from a template. Creating this resource LAUNCHES a job (a side
# effect) and, by default, waits for it to finish. It records a point-in-time
# execution - it does not manage ongoing state. All launch inputs are ForceNew,
# so changing them launches a new job rather than editing the existing one.
resource "stackweaver_ansible_job" "deploy" {
  job_template_id = stackweaver_ansible_job_template.deploy.id

  extra_vars = jsonencode({
    environment = "production"
    version     = "1.4.2"
  })

  wait_for_completion = true
}
