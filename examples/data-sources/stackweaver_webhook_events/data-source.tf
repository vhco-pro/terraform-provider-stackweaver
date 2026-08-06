data "stackweaver_webhook_events" "recent" {
  organization = "my-org-name"
}

output "failed_deliveries" {
  value = [
    for e in data.stackweaver_webhook_events.recent.events : e.id
    if e.status == "failed"
  ]
}
