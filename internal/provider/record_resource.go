package provider

import (
	"context"
	"fmt"
	"terraform-provider-ddnsnow/pkg/ddnsnow"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &recordResource{}
	_ resource.ResourceWithConfigure = &recordResource{}
)

// NewRecordResource is a helper function to simplify the provider implementation.
func NewRecordResource() resource.Resource {
	return &recordResource{}
}

// recordResource is the resource implementation.
type recordResource struct {
	client ddnsnow.Client
}

// Metadata returns the resource type name.
func (r *recordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_record"
}

// Schema defines the schema for the resource.
func (r *recordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Required: true,
			},
			"value": schema.StringAttribute{
				Required: true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *recordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan recordResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	record := ddnsnow.Record{
		Type:  ddnsnow.RecordType(plan.Type.ValueString()),
		Value: plan.Value.ValueString(),
	}

	// Create new record
	err := r.client.CreateRecord(record)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating record",
			"Could not create record, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.Type = types.StringValue(string(record.Type))
	plan.Value = types.StringValue(record.Value)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *recordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state recordResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed record value from DDNS Now
	record, err := r.client.GetRecord(ddnsnow.Record{
		Type:  ddnsnow.RecordType(state.Type.ValueString()),
		Value: state.Value.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading DDNS Now Record",
			"Could not read DDNS Now record type "+state.Type.ValueString()+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	state.Type = types.StringValue(string(record.Type))
	state.Value = types.StringValue(record.Value)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *recordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from state
	var state recordResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Retrieve values from plan
	var plan recordResourceModel
	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	oldRecord := ddnsnow.Record{
		Type:  ddnsnow.RecordType(state.Type.ValueString()),
		Value: state.Value.ValueString(),
	}
	newRecord := ddnsnow.Record{
		Type:  ddnsnow.RecordType(plan.Type.ValueString()),
		Value: plan.Value.ValueString(),
	}

	// Update existing record
	err := r.client.UpdateRecord(oldRecord, newRecord)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating DDNS Now Record",
			"Could not update record, unexpected error: "+err.Error(),
		)
		return
	}

	// Fetch updated items from GetRecord as UpdateRecord items are not
	// populated.
	_, err = r.client.GetRecord(newRecord)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading DDNS Now Record",
			"Could not read DDNS Now record type "+plan.Type.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update resource state with updated record
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *recordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state recordResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing record
	err := r.client.DeleteRecord(ddnsnow.Record{
		Type:  ddnsnow.RecordType(state.Type.ValueString()),
		Value: state.Value.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting DDNS Now Record",
			"Could not delete record, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *recordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(ddnsnow.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected ddnsnow.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// recordResourceModel maps the resource schema data.
type recordResourceModel struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}
