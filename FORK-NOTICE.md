# Fork notice

`terraform-provider-stackweaver` is a **standalone provider derived from**
[`hashicorp/terraform-provider-tfe`](https://github.com/hashicorp/terraform-provider-tfe),
maintained by VH & Co BV for the [Stackweaver](https://stackweaver.io) platform. It is **not** an
official HashiCorp product and is not affiliated with or endorsed by HashiCorp.

- **License:** Mozilla Public License 2.0 (MPL-2.0), inherited from upstream and unchanged. Upstream
  copyright headers (`Copyright (c) HashiCorp, Inc.`) are preserved on all forked files; see
  [`LICENSE`](./LICENSE).
- **Baseline:** upstream release **v0.79.0** (the tracked watermark).
- **Standalone history, diff-based sync.** This repository has **its own git history** (a single
  initial commit at the v0.79.0 baseline) - it is not a mirror of upstream's history. `upstream`
  (`hashicorp/terraform-provider-tfe`) is a **fetch-only** remote used only for comparison: relevant
  upstream changes are found by **targeted diff** of the supported files between release tags and
  applied as native commits by the sync agent - **never** by `git merge`. This keeps the provider a
  first-class Stackweaver project while staying fully compatible with upstream.
- **Status:** work in progress. Built spec-first - the specification under `spec/` (and plan under
  `plan/`) is written and reviewed before implementation.

The provider targets Stackweaver's TFE-compatible API. It exposes `stackweaver_*` resources with
`tfe_*` aliases for a drop-in migration path, and adds Stackweaver-native resources that have no TFE
equivalent.
