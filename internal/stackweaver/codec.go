// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"encoding/json"
	"errors"
	"fmt"
)

// The native API envelope is mixed: most Ansible resources speak JSON:API (a
// { "data": { "type", "id", "attributes", "relationships" } } envelope, with
// dash-cased attribute keys), while a handful of endpoints (ansible_config,
// notification template/attachment, the VCS listing + playbook-file discovery
// endpoints) are plain JSON. The client therefore provides BOTH codecs and each
// service method declares which one it uses, per that resource's spec wire
// contract.

// jsonAPICodec encodes/decodes the JSON:API envelope used by most native
// Ansible resources.
type jsonAPICodec struct{}

// Marshal wraps an attributes value into a JSON:API request body of the shape
//
//	{"data":{"type":<resourceType>,"attributes":{...}}}
//
// It is the encoder for attributes-only resources. Resources that also carry
// relationships (e.g. ansible_playbook, whose project and vcs-connection are
// relationships) build their own envelope struct and marshal it directly.
func (c *jsonAPICodec) Marshal(resourceType string, attributes any) ([]byte, error) {
	payload := map[string]any{
		"data": map[string]any{
			"type":       resourceType,
			"attributes": attributes,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("stackweaver: encode json:api request: %w", err)
	}
	return body, nil
}

// Unmarshal decodes a JSON:API response body into out. It unwraps the top-level
// {"data":...} envelope and decodes the resource object (id/type/attributes/
// relationships) into out, so out mirrors that object's shape. out may be a
// single struct (single-resource responses) or a slice (collection responses),
// since the "data" member is a struct or an array respectively.
func (c *jsonAPICodec) Unmarshal(body []byte, out any) error {
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("stackweaver: decode json:api envelope: %w", err)
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return errors.New("stackweaver: json:api response has no data member")
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("stackweaver: decode json:api resource: %w", err)
	}
	return nil
}

// plainJSONCodec encodes/decodes the plain-JSON endpoints (no JSON:API
// envelope).
type plainJSONCodec struct{}

// Marshal encodes v as a plain-JSON request body.
func (c *plainJSONCodec) Marshal(v any) ([]byte, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("stackweaver: encode json request: %w", err)
	}
	return body, nil
}

// Unmarshal decodes a plain-JSON response body into out.
func (c *plainJSONCodec) Unmarshal(body []byte, out any) error {
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("stackweaver: decode json response: %w", err)
	}
	return nil
}
