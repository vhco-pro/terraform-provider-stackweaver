---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_ansible_collections"
subcategory: "Ansible"
description: |-
  Lists the Ansible Galaxy collections pre-installed on the Stackweaver runner image.
---

# stackweaver_ansible_collections (Data Source)

Use this data source to list the Ansible Galaxy collections pre-installed on the Stackweaver runner
image. It takes no inputs. Galaxy search is not yet available; this data source reports only what ships
on the runner.

This is a native Stackweaver data source with no `terraform-provider-tfe` equivalent.

## Example Usage

```hcl
data "stackweaver_ansible_collections" "installed" {}

output "collection_names" {
  value = [for c in data.stackweaver_ansible_collections.installed.collections : c.name]
}
```

## Argument Reference

This data source takes no arguments.

## Attributes Reference

The following attributes are exported:

* `id` - Synthesized constant identifier (`pre-installed`).
* `collections` - Pre-installed collections on the runner. Each element documented below.

The `collections` block contains:

* `name` - Fully-qualified collection name, e.g. `amazon.aws`.
* `namespace` - Collection namespace, e.g. `amazon`.
* `version` - Installed version; `latest` for pre-installed collections.
* `description` - Human-readable description.
* `source` - Origin of the collection; `pre-installed` for this data source.
