data "stackweaver_ansible_inventory_syncs" "history" {
  inventory_id = "inv-0123456789abcdef"
}

output "last_sync_status" {
  value = try(data.stackweaver_ansible_inventory_syncs.history.syncs[0].status, null)
}
