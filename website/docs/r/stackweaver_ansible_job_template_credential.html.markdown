---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_ansible_job_template_credential"
subcategory: "Ansible"
description: |-
  Attaches a credential to an Ansible job template.
---

# stackweaver_ansible_job_template_credential

Native Stackweaver resource — there is no `terraform-provider-tfe` equivalent.

Attaches one Ansible credential to a job template's multi-credential set, enforcing the
AWX one-credential-per-type rule. This is a pure association with no in-place update:
changing either the job template or the credential detaches and re-attaches.

Use this resource to manage the multi-credential association on a job template. The single
legacy machine credential is managed instead by the `credential_id` argument on
[`stackweaver_ansible_job_template`](stackweaver_ansible_job_template.html).

## Example Usage

Basic usage:

```hcl
resource "stackweaver_ansible_job_template" "deploy" {
  organization = "my-org-name"
  name         = "deploy-web"
  playbook_id  = stackweaver_ansible_playbook.site.id
  inventory_id = stackweaver_ansible_inventory.production.id
}

resource "stackweaver_ansible_job_template_credential" "vault" {
  job_template_id = stackweaver_ansible_job_template.deploy.id
  credential_id   = stackweaver_ansible_credential.vault.id
}
```

## Argument Reference

The following arguments are supported:

* `job_template_id` - (Required) ID of the job template. Changing this re-attaches the credential.
* `credential_id` - (Required) ID of the credential to attach. Must belong to the template's organization. Changing this re-attaches.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The synthetic composite identifier, `<job_template_id>/<credential_id>`.
* `credential_type` - Type of the attached credential (the per-type uniqueness key).
* `credential_name` - Name of the attached credential.

## Import

Ansible job template credential attachments can be imported using the composite ID
`<job_template_id>/<credential_id>`. For example:

```shell
terraform import stackweaver_ansible_job_template_credential.vault 5e7f8a90-1b2c-3d4e-5f60-7a8b9c0d1e2f/9a8b7c6d-5e4f-3a2b-1c0d-9e8f7a6b5c4d
```
