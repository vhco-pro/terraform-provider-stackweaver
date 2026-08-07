---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_ansible_credential"
subcategory: "Ansible"
description: |-
  Manages an Ansible credential - a named, typed secret bundle.
---

# stackweaver_ansible_credential

Manages an Ansible credential: a named, typed secret bundle scoped to an organization (and optionally
narrowed to a project) that job templates, jobs, and inventory sources attach for host access, SCM
access, vault decryption, or cloud-inventory authentication.

This is a native Stackweaver resource with no `terraform-provider-tfe` equivalent. It is the intended
home for an eventual `stackweaver_ssh_key`-style face over the `ssh` credential type.

## Write-only secret behavior

All secret material is accepted on write and **never returned by the API**. Terraform state holds the
value the practitioner wrote - that is the only place it exists after apply. The provider does not
reconcile secret attributes from the read response, so an omitted server value never produces a
perpetual diff. Rotating a secret is a normal in-place update: send the new value.

For four of the secrets the API surfaces a computed presence boolean - `has_ssh_private_key`,
`has_password`, `has_vault_password`, and `has_become_password` - so you can confirm a secret is
stored without reading it back. The AWS, Azure, and GCP secrets have no presence readback; their
presence can only be tracked from state.

Which secret attributes apply depends on `credential_type`:

* `ssh` and `machine-ssh` - `ssh_private_key`, `ssh_passphrase`, `password`, `become_password`, plus
  `username`, `ssh_port`, and `ssh_become_user`.
* `scm` - `username` and `password` (or `ssh_private_key`) for source-control access.
* `vault` - `vault_password`.
* `aws` - `aws_access_key_id` and `aws_secret_access_key`.
* `azure` - `azure_tenant_id`, `azure_client_id`, and `azure_client_secret`.
* `gcp` - `gcp_service_account`.
* `vmware` - `username` and `password`.

## Example Usage

Basic usage - an SSH credential fed from a variable:

```hcl
variable "ssh_private_key" {
  type      = string
  sensitive = true
}

resource "stackweaver_ansible_credential" "web_ssh" {
  organization    = "my-org-name"
  name            = "web-ssh"
  description     = "SSH key for the web fleet"
  credential_type = "ssh"
  username        = "ansible"
  ssh_private_key = var.ssh_private_key
}
```

An AWS credential for dynamic-inventory discovery:

```hcl
variable "aws_access_key_id" {
  type      = string
  sensitive = true
}

variable "aws_secret_access_key" {
  type      = string
  sensitive = true
}

resource "stackweaver_ansible_credential" "aws" {
  organization          = "my-org-name"
  name                  = "aws-inventory"
  credential_type       = "aws"
  aws_access_key_id     = var.aws_access_key_id
  aws_secret_access_key = var.aws_secret_access_key
}
```

## Argument Reference

The following arguments are supported:

* `organization` - (Required) Name of the owning organization. Changing this forces a new credential.
* `name` - (Required) Name of the credential, unique within the organization.
* `credential_type` - (Required) Credential type: one of `ssh`, `scm`, `vault`, `machine-ssh`, `aws`,
  `azure`, `gcp`, `vmware`. Immutable - changing it forces a new credential.
* `project_id` - (Optional) ID of the project to narrow the credential to. Omit for an org-scoped
  credential (resolves to the organization's default project). Not returned by the API, so it is
  tracked from configuration only.
* `description` - (Optional) Human-readable description of the credential.
* `username` - (Optional) Username for host or SCM access.
* `azure_tenant_id` - (Optional) Azure tenant ID (not a secret).
* `azure_client_id` - (Optional) Azure client ID (not a secret).
* `ssh_port` - (Optional) SSH port for host access. Defaults to `22`.
* `ssh_become_user` - (Optional) User to become for privilege escalation. Defaults to `root`.
* `ssh_private_key` - (Optional, Sensitive) SSH private key. Write-only - never read back.
* `ssh_passphrase` - (Optional, Sensitive) Passphrase for the SSH private key. Write-only.
* `password` - (Optional, Sensitive) Password for host or SCM access. Write-only.
* `vault_password` - (Optional, Sensitive) Ansible Vault password. Write-only.
* `become_password` - (Optional, Sensitive) Privilege-escalation (sudo) password. Write-only.
* `aws_access_key_id` - (Optional, Sensitive) AWS access key ID. Write-only, with no presence readback.
* `aws_secret_access_key` - (Optional, Sensitive) AWS secret access key. Write-only, with no presence
  readback.
* `azure_client_secret` - (Optional, Sensitive) Azure client secret. Write-only, with no presence
  readback.
* `gcp_service_account` - (Optional, Sensitive) GCP service account JSON. Write-only, with no presence
  readback.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The credential ID.
* `has_ssh_private_key` - Whether an SSH private key is stored.
* `has_password` - Whether a password is stored.
* `has_vault_password` - Whether a vault password is stored.
* `has_become_password` - Whether a become password is stored.

## Import

Ansible credentials can be imported using their ID. Secret attributes cannot be imported (the API
never returns them); set them in configuration after import. For example:

```shell
terraform import stackweaver_ansible_credential.web_ssh 3f8c1e2a-9b4d-4f7a-8c1e-2a9b4d4f7a8c
```
