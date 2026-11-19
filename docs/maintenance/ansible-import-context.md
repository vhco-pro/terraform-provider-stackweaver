# Maintenance: preserve organization context during Ansible imports

## Definition

Outcome: import existing Ansible inventories, credentials and job templates without losing the required immutable organization attribute.

Preserved contract: an unqualified import ID identifies the same remote object; reads and writes use the existing resource APIs. No remote resource is created, replaced or deleted by this fix. The optional inventory project remains unavailable in the native read response and must be reviewed separately.

Task base: `024673286aad926d1acb3e828bc9cf74f2fd8186`.

Affected paths: `internal/provider/ansible_import_context.go`, its test, the three Ansible inventory/credential/job-template resource files, their resource documentation, and `.github/workflows/ci.yml`.

Verification inputs: `go.mod`, `go.sum`, the affected provider schemas and import methods, the existing native client, and the CI unit-test selection.

Proposed change: preserve the organization selected by provider configuration during import, or accept an explicit `<organization>/<resource-id>` import identifier. Reject identifiers without organization context or with malformed separators before reading or writing a remote resource. Keep the raw resource ID in state.

Required checks: observe the missing-context regression failing before the fix; pass the table-driven tests for all three resource types, including malformed input; run `go build ./...`, `go vet ./...`, native client tests and the CI-selected provider tests. Review a read-only import plan against an existing owned inventory; its action must be no-op, with its existing identity and organization preserved. No apply is part of this check.

## Evidence

The new tests failed on the original importers because organization remained null and missing context was accepted. The repaired table passed all 18 cases. Build, vet and CI-selected tests passed locally.

A locally built fixture-only provider produced a no-op import plan against an existing inventory; the published 0.4.0 provider had instead proposed replacement for its null organization. No plan was applied and the production provider was unchanged. Deployment-specific identifiers and protected plans remain in the private consuming repository/custody.

The broader non-acceptance unit run passed with `-skip '^TestAcc|^TestConfigureEnvOrganization$'`. The excluded upstream configuration test requires CLI credentials. Earlier attempts without that prerequisite, and a sandbox-restricted DNS attempt, failed and are retained locally. Live acceptance tests were not run.

## Reviews

Independent review pending.

## State

Review pending. No provider release or deployment authorized by this record.
