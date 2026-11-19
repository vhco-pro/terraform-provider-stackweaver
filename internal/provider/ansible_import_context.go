// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// These resources require the organization's name, which the native read API
// does not return. Preserve caller-selected context during import so a missing
// immutable attribute cannot turn adoption into replacement.
func importAnsibleOrganization(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse, defaultOrganization string) {
	organization := strings.TrimSpace(defaultOrganization)
	id := req.ID
	if strings.Contains(id, "/") {
		var explicit string
		explicit, id, _ = strings.Cut(id, "/")
		organization = strings.TrimSpace(explicit)
	}
	if organization == "" || strings.TrimSpace(id) == "" || strings.Contains(id, "/") {
		resp.Diagnostics.AddError("Organization required for Ansible import",
			"Use <organization>/<resource-id>, or configure the provider organization and import by resource ID.")
		return
	}
	req.ID = id
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization"), organization)...)
}
