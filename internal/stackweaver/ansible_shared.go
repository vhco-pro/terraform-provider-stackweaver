// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

// This file holds the JSON:API request-envelope helpers shared by the native
// Ansible services that carry relationships (credentials, inventory sources).
// The reference playbook service keeps its own local envelope struct; these are
// the generalized equivalents so later services don't each redeclare one.

// jsonAPIRequest is the top-level JSON:API write envelope ({"data":{...}}).
type jsonAPIRequest struct {
	Data jsonAPIRequestData `json:"data"`
}

// jsonAPIRequestData is the resource object sent on create/update. Attributes
// carry only the keys the caller sets (so the server applies its defaults);
// relationships are included only when a linkage is supplied.
type jsonAPIRequestData struct {
	Type          string                         `json:"type"`
	Attributes    map[string]any                 `json:"attributes"`
	Relationships map[string]jsonAPIRelationship `json:"relationships,omitempty"`
}

// setIfNotEmpty adds key=val to m only when val is non-empty, so omitted inputs
// fall through to the server's defaults instead of overwriting them with "".
func setIfNotEmpty(m map[string]any, key, val string) {
	if val != "" {
		m[key] = val
	}
}
