#!/usr/bin/env bash
# Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0
#
# Acceptance run for stackweaver_organization and the organization data sources, against a live
# Stackweaver stack through dev_overrides. Each check names the spec criterion it proves:
#   R<n>  spec/resources/tfe_organization.md        acceptance criterion n
#   D<n>  spec/data-sources/tfe_organization.md     acceptance criterion n
#   L<n>  spec/data-sources/tfe_organizations.md    acceptance criterion n
#   RT    the resource spec's runtime criterion
#
# Environment:
#   TFE_TOKEN     (required) a USER token; an organization-scoped token cannot create organizations
#   TFE_HOSTNAME  (optional) default stackweaver.vhco.pro; the provider needs HTTPS
#   PROVIDER_BIN  (optional) directory holding the built provider binary; default: builds into a temp dir
#
# Usage: test/fixtures/organization/run.sh
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/../../.." && pwd)"
: "${TFE_TOKEN:?TFE_TOKEN must be a user token}"
export TFE_HOSTNAME="${TFE_HOSTNAME:-stackweaver.vhco.pro}"
BASE="https://$TFE_HOSTNAME/api/v2"
USER_TOKEN="$TFE_TOKEN"

pass() { echo "PASS $*" >&2; }
fail() { echo "FAIL $*" >&2; exit 1; }
check() { local id="$1" desc="$2"; shift 2; if "$@"; then pass "$id $desc"; else fail "$id $desc"; fi; }
status() { curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $USER_TOKEN" "$BASE$1"; }

WORK="$(mktemp -d)"
umask 077
if [ -z "${PROVIDER_BIN:-}" ]; then
  PROVIDER_BIN="$WORK/bin"
  (cd "$ROOT" && go build -o "$PROVIDER_BIN/terraform-provider-stackweaver" .)
fi
cat > "$WORK/terraformrc" <<EOF
provider_installation {
  dev_overrides {
    "vhco-pro/stackweaver" = "$PROVIDER_BIN"
  }
  direct {}
}
EOF
export TF_CLI_CONFIG_FILE="$WORK/terraformrc" TF_IN_AUTOMATION=1
cp -r "$HERE"/. "$WORK/fx"
FX="$WORK/fx"
RUN_ID="$(date +%s)-$RANDOM"
NAME="pv-org-$RUN_ID"
TF() { local dir="$1"; shift; terraform -chdir="$dir" "$@"; }

cleanup() {
  local rc=$?
  TFE_TOKEN="$USER_TOKEN" TF "$FX/alias" destroy -auto-approve -input=false -var "run_id=$RUN_ID" >/dev/null 2>&1 || true
  TFE_TOKEN="$USER_TOKEN" TF "$FX" destroy -auto-approve -input=false -var "run_id=$RUN_ID" >/dev/null 2>&1 || true
  rm -rf "$WORK"
  exit "$rc"
}
trap cleanup EXIT

# ---- create (R1, D1, D2, L1, L3, RT) ------------------------------------------------------------
TF "$FX" apply -auto-approve -input=false -no-color -var "run_id=$RUN_ID" >/dev/null
ORG="$(TF "$FX" output -json organization)"
attr() { jq -r ".$1" <<<"$ORG"; }
check R1 "name, email and id round-trip" test "$(attr name)/$(attr email)/$(attr id)" = "$NAME/provider-acceptance@example.test/$NAME"
check R1 "collaborator_auth_policy round-trips" test "$(attr collaborator_auth_policy)" = password
check R1 "cost_estimation_enabled round-trips" test "$(attr cost_estimation_enabled)" = true
check R1 "allow_force_delete_workspaces round-trips" test "$(attr allow_force_delete_workspaces)" = true
check R1 "assessments_enforced round-trips" test "$(attr assessments_enforced)" = true
check R1 "speculative_plan_management_enabled round-trips" test "$(attr speculative_plan_management_enabled)" = false
check R1 "aggregated_commit_status_enabled round-trips" test "$(attr aggregated_commit_status_enabled)" = true
DEFAULT_PROJECT="$(attr default_project_id)"
check R1 "default_project_id is set" test -n "$DEFAULT_PROJECT" -a "$DEFAULT_PROJECT" != null

DS="$(TF "$FX" output -json ds)"
ds() { jq -r ".$1" <<<"$DS"; }
check D1 "external_id is a UUID" grep -qE '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' <<<"$(ds external_id)"
check D1 "email and default_project_id match the resource" test "$(ds email)/$(ds default_project_id)" = "provider-acceptance@example.test/$DEFAULT_PROJECT"
check D1 "policy flags match the resource" test "$(ds cost_estimation_enabled)/$(ds assessments_enforced)/$(ds speculative_plan_management_enabled)/$(ds aggregated_commit_status_enabled)" = "true/true/false/true"
check D2 "the tfe_organization alias reads the same values" test "$(jq -cS 'del(.id)' <<<"$DS")" = "$(TF "$FX" output -json ds_alias | jq -cS 'del(.id)')"
check L1 "names contains the new organization" jq -e --arg n "$NAME" 'index($n) != null' <<<"$(TF "$FX" output -json list_names)" >/dev/null
check L1 "ids maps it to its external_id" test "$(TF "$FX" output -json list_ids | jq -r --arg n "$NAME" '.[$n]')" = "$(ds external_id)"
check L3 "the tfe_organizations alias returns the same ids" test "$(TF "$FX" output -json list_ids | jq -cS .)" = "$(TF "$FX" output -json list_alias_ids | jq -cS .)"

PROJECTS="$(curl -sS -H "Authorization: Bearer $USER_TOKEN" "$BASE/organizations/$NAME/projects")"
check RT "default_project_id is the project named default" test "$(jq -r --arg id "$DEFAULT_PROJECT" '.data[] | select(.id == $id) | .attributes.name' <<<"$PROJECTS")" = default
check RT "the creator created a workspace in it" test -n "$(TF "$FX" output -raw workspace_id)"

# ---- no drift (R2) ------------------------------------------------------------------------------
check R2 "re-plan shows no changes" TF "$FX" plan -detailed-exitcode -input=false -no-color -var "run_id=$RUN_ID" >/dev/null

# ---- in-place update (R3) -----------------------------------------------------------------------
UPDATE_VARS=(-var "run_id=$RUN_ID" -var "email=updated@example.test" -var "assessments_enforced=false")
TF "$FX" plan -input=false -no-color -out="$WORK/update.plan" "${UPDATE_VARS[@]}" >/dev/null
ACTIONS="$(TF "$FX" show -json "$WORK/update.plan" | jq -c '[.resource_changes[] | select(.address == "stackweaver_organization.test") | .change.actions[]]')"
check R3 "changing email and assessments_enforced is an in-place update" test "$ACTIONS" = '["update"]'
TF "$FX" apply -auto-approve -input=false -no-color "$WORK/update.plan" >/dev/null
check R3 "re-plan after the update shows no changes" TF "$FX" plan -detailed-exitcode -input=false -no-color "${UPDATE_VARS[@]}" >/dev/null

# ---- alias resource (R4) ------------------------------------------------------------------------
TF "$FX/alias" apply -auto-approve -input=false -no-color -var "run_id=$RUN_ID" >/dev/null
ALIAS_NAME="$(TF "$FX/alias" output -raw name)"
check R4 "tfe_organization applies and re-plans clean" TF "$FX/alias" plan -detailed-exitcode -input=false -no-color -var "run_id=$RUN_ID" >/dev/null
TF "$FX/alias" destroy -auto-approve -input=false -no-color -var "run_id=$RUN_ID" >/dev/null
check R4 "the alias organization is gone after destroy (404)" test "$(status "/organizations/$ALIAS_NAME")" = 404

# ---- import (R5) and provider-default read (D3) -------------------------------------------------
IMPORT_VARS=(-var "name=$NAME" -var "email=updated@example.test" -var "assessments_enforced=false")
TF "$FX/import" import -input=false -no-color "${IMPORT_VARS[@]}" stackweaver_organization.imported "$NAME" >/dev/null
TF "$FX/import" plan -input=false -no-color -out="$WORK/import.plan" "${IMPORT_VARS[@]}" >/dev/null
IMPORT_PLAN="$(TF "$FX/import" show -json "$WORK/import.plan")"
check R5 "an imported organization plans no changes" test "$(jq -c '[.resource_changes[] | select(.address == "stackweaver_organization.imported") | .change.actions[]]' <<<"$IMPORT_PLAN")" = '["no-op"]'
check D3 "with name omitted, the data source reads the provider's organization" test "$(jq -r '.planned_values.outputs.provider_default_name.value' <<<"$IMPORT_PLAN")" = "$NAME"
TF "$FX/import" state rm stackweaver_organization.imported >/dev/null

# ---- organization-scoped token (R7, L2) ---------------------------------------------------------
TF "$FX" output -raw org_token > "$WORK/org-token"
export TFE_TOKEN="$(cat "$WORK/org-token")"
TF "$FX/refusal" apply -auto-approve -input=false -no-color -var "run_id=$RUN_ID" -target=data.stackweaver_organizations.scoped >/dev/null 2>&1
check L2 "an organization-scoped token lists exactly its own organization" test "$(TF "$FX/refusal" output -json scoped_names | jq -c .)" = "[\"$NAME\"]"
REFUSAL_OUT="$(TF "$FX/refusal" apply -auto-approve -input=false -no-color -var "run_id=$RUN_ID" 2>&1 || true)"
check R7 "creating with an organization-scoped token is refused" grep -q "organization-scoped tokens cannot create organizations" <<<"$REFUSAL_OUT"
check R7 "and creates nothing" test "$(status "/organizations/pv-refused-$RUN_ID")" = 404
export TFE_TOKEN="$USER_TOKEN"

# ---- destroy (R6) -------------------------------------------------------------------------------
TF "$FX" destroy -auto-approve -input=false -no-color "${UPDATE_VARS[@]}" >/dev/null
check R6 "the organization is gone after destroy (404)" test "$(status "/organizations/$NAME")" = 404

echo "ALL PASS (run $RUN_ID)"
