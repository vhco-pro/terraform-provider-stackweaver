// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestAnsibleImportOrganizationContext(t *testing.T) {
	ctx := context.Background()
	factories := map[string]func(string) resource.ResourceWithImportState{
		"inventory": func(org string) resource.ResourceWithImportState {
			return &resourceStackweaverAnsibleInventory{config: ConfiguredClient{Organization: org}}
		},
		"credential": func(org string) resource.ResourceWithImportState {
			return &resourceStackweaverAnsibleCredential{config: ConfiguredClient{Organization: org}}
		},
		"job_template": func(org string) resource.ResourceWithImportState {
			return &resourceStackweaverAnsibleJobTemplate{config: ConfiguredClient{Organization: org}}
		},
	}
	for kind, factory := range factories {
		for _, tc := range []struct {
			name, id, defaultOrg, wantID, wantOrg string
			wantError                             bool
		}{
			{name: "provider default", id: "resource-id", defaultOrg: "example", wantID: "resource-id", wantOrg: "example"},
			{name: "qualified scope", id: "selected/resource-id", defaultOrg: "example", wantID: "resource-id", wantOrg: "selected"},
			{name: "qualified without default", id: "selected/resource-id", wantID: "resource-id", wantOrg: "selected"},
			{name: "missing organization", id: "resource-id", wantError: true},
			{name: "empty explicit organization", id: "/resource-id", defaultOrg: "example", wantError: true},
			{name: "empty resource", id: "selected/", wantError: true},
		} {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				instance := factory(tc.defaultOrg)
				schemaResponse := resource.SchemaResponse{}
				instance.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
				object := schemaResponse.Schema.Type().TerraformType(ctx).(tftypes.Object)
				values := map[string]tftypes.Value{}
				for name, kind := range object.AttributeTypes {
					values[name] = tftypes.NewValue(kind, nil)
				}
				response := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResponse.Schema, Raw: tftypes.NewValue(object, values)}}
				instance.ImportState(ctx, resource.ImportStateRequest{ID: tc.id}, &response)
				if response.Diagnostics.HasError() != tc.wantError {
					t.Fatalf("diagnostics: %v", response.Diagnostics)
				}
				if tc.wantError {
					return
				}
				var id, org types.String
				if diag := response.State.GetAttribute(ctx, path.Root("id"), &id); diag.HasError() {
					t.Fatal(diag)
				}
				if diag := response.State.GetAttribute(ctx, path.Root("organization"), &org); diag.HasError() {
					t.Fatal(diag)
				}
				if id.ValueString() != tc.wantID || org.IsNull() || org.ValueString() != tc.wantOrg {
					t.Fatalf("imported id=%q organization=%v, want id=%q organization=%q", id.ValueString(), org, tc.wantID, tc.wantOrg)
				}
			})
		}
	}
}
