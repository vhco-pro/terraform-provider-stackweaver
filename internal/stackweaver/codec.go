// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

// The native API envelope is mixed (plan.md §4): most Ansible resources speak
// JSON:API (a { "data": { "type", "id", "attributes" } } envelope, snake- or
// dash-cased), while a handful of endpoints (ansible_config, notification
// template/attachment, the VCS listing + playbook-file discovery endpoints) are
// plain JSON. The client therefore provides BOTH codecs and each service method
// declares which one it uses, per that resource's spec wire contract.
//
// These are scaffold stubs: they define the codec seam so native waves can fill
// in encode/decode without reshaping the client. Real bodies are added in
// T-native-0.

// jsonAPICodec encodes/decodes the JSON:API envelope used by most native
// Ansible resources.
type jsonAPICodec struct{}

// Marshal wraps a resource attributes value into a JSON:API request body.
func (c *jsonAPICodec) Marshal(resourceType string, attributes any) ([]byte, error) {
	return nil, ErrNotImplemented
}

// Unmarshal decodes a JSON:API response body into out.
func (c *jsonAPICodec) Unmarshal(body []byte, out any) error {
	return ErrNotImplemented
}

// plainJSONCodec encodes/decodes the plain-JSON endpoints (no JSON:API
// envelope).
type plainJSONCodec struct{}

// Marshal encodes v as a plain-JSON request body.
func (c *plainJSONCodec) Marshal(v any) ([]byte, error) {
	return nil, ErrNotImplemented
}

// Unmarshal decodes a plain-JSON response body into out.
func (c *plainJSONCodec) Unmarshal(body []byte, out any) error {
	return ErrNotImplemented
}
