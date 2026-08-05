# Fork notice

`terraform-provider-stackweaver` is a fork of
[`hashicorp/terraform-provider-tfe`](https://github.com/hashicorp/terraform-provider-tfe),
maintained by VH & Co BV for the [Stackweaver](https://stackweaver.io) platform. It is **not** an
official HashiCorp product and is not affiliated with or endorsed by HashiCorp.

- **License:** Mozilla Public License 2.0 (MPL-2.0), inherited from upstream and unchanged. Upstream
  copyright headers (`Copyright (c) HashiCorp, Inc.`) are preserved on all forked files; see
  [`LICENSE`](./LICENSE).
- **Fork baseline:** upstream `v0.79.0`. Upstream changes are merged in from the `upstream` remote
  (`hashicorp/terraform-provider-tfe`) against a tracked watermark.
- **Status:** work in progress. This provider is built spec-first — the specification under `spec/`
  is written and reviewed before implementation. Until a tagged release exists, treat `main` as the
  upstream baseline plus in-progress Stackweaver changes.

The provider targets Stackweaver's TFE-compatible API. It exposes `stackweaver_*` resources with
`tfe_*` aliases for a drop-in migration path, and adds Stackweaver-native resources that have no TFE
equivalent.
