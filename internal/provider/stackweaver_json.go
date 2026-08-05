// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Native Stackweaver resources carry a few jsonb fields (notification-template
// config, schedule config, job extra_vars). There is no jsontypes dependency in
// this provider, so these are modeled as JSON-string attributes. The helpers here
// keep them drift-free: the plan value is echoed on write, and a refresh only
// rewrites state when the server value is *semantically* different (so key
// reordering by the server never produces a perpetual diff).

// jsonStringToMap parses a JSON object string into a map. An empty/blank string
// yields a nil map (send nothing / let the server default apply).
func jsonStringToMap(s string) (map[string]interface{}, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	return m, nil
}

// mapToJSONString marshals a map to its canonical JSON string ("{}" for empty).
func mapToJSONString(m map[string]interface{}) string {
	if len(m) == 0 {
		return "{}"
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// jsonSemanticEqual reports whether two JSON strings encode the same value,
// ignoring key order and whitespace.
func jsonSemanticEqual(a, b string) bool {
	var av, bv interface{}
	if err := json.Unmarshal([]byte(a), &av); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &bv); err != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}

// planJSONState is the value to store for a jsonb attribute after a create/update:
// echo the configured plan value verbatim when set (so the result matches the
// plan exactly), otherwise fall back to the server's canonical rendering (for the
// Optional+Computed "unset, server-defaulted" case).
func planJSONState(configured types.String, serverMap map[string]interface{}) types.String {
	if !configured.IsNull() && !configured.IsUnknown() {
		return configured
	}
	return types.StringValue(mapToJSONString(serverMap))
}

// refreshJSONState is the value to store for a jsonb attribute during a Read:
// keep the prior state string when it is semantically equal to the server value
// (no drift from key reordering), otherwise adopt the server's canonical value.
func refreshJSONState(prior types.String, serverMap map[string]interface{}) types.String {
	server := mapToJSONString(serverMap)
	if !prior.IsNull() && !prior.IsUnknown() && jsonSemanticEqual(prior.ValueString(), server) {
		return prior
	}
	return types.StringValue(server)
}
