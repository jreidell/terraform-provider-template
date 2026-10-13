package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/jreidell/terraform-provider-template/internal/uuidclient"
)

var (
	_ resource.Resource               = &ServerResource{}
	_ resource.ResourceWithConfigure  = &ServerResource{}
	_ resource.ResourceWithModifyPlan = &ServerResource{}
)

// NewServerResource returns the template_server resource.
func NewServerResource() resource.Resource {
	return &ServerResource{}
}

// ServerResource manages a set of generated UUIDs. The delay attributes
// simulate a slow API so long-running operations can be exercised.
type ServerResource struct {
	client *uuidclient.Client
}

// ServerResourceModel maps the resource schema to Go types.
type ServerResourceModel struct {
	ID            types.String `tfsdk:"id"`
	UUIDCount     types.Int64  `tfsdk:"uuid_count"`
	UUIDs         types.List   `tfsdk:"uuids"`
	CreationDelay types.Int64  `tfsdk:"creation_delay"`
	UpdateDelay   types.Int64  `tfsdk:"update_delay"`
	DeletionDelay types.Int64  `tfsdk:"deletion_delay"`
}

func (r *ServerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (r *ServerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Generates a set of UUIDs and keeps them in state. " +
			"The UUIDs are regenerated only when `uuid_count` changes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the server, generated on creation.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uuid_count": schema.Int64Attribute{
				MarkdownDescription: fmt.Sprintf("Number of UUIDs to generate, from 1 to %d.", uuidclient.MaxCount),
				Required:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, uuidclient.MaxCount),
				},
			},
			"uuids": schema.ListAttribute{
				MarkdownDescription: "The generated UUIDs.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"creation_delay": delayAttribute("creating"),
			"update_delay":   delayAttribute("updating"),
			"deletion_delay": delayAttribute("deleting"),
		},
	}
}

func delayAttribute(operation string) schema.Int64Attribute {
	return schema.Int64Attribute{
		MarkdownDescription: "Seconds to wait before " + operation + " the server, simulating a slow API. Defaults to `0`.",
		Optional:            true,
		Computed:            true,
		Default:             int64default.StaticInt64(0),
		Validators: []validator.Int64{
			int64validator.AtLeast(0),
		},
	}
}

func (r *ServerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// ProviderData is nil until the provider has been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*uuidclient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *uuidclient.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// ModifyPlan keeps the existing UUIDs in the plan unless uuid_count changes.
// Without it every update would show uuids as "known after apply".
func (r *ServerResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Nothing to carry over when creating or destroying.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var plan, state ServerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if plan.UUIDCount.Equal(state.UUIDCount) {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("uuids"), state.UUIDs)...)
	}
}

func (r *ServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ServerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := wait(ctx, plan.CreationDelay); err != nil {
		resp.Diagnostics.AddError("Server Creation Interrupted", err.Error())

		return
	}

	ids, err := r.client.Generate(ctx, 1)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Server", "Generating the server ID failed: "+err.Error())

		return
	}

	plan.ID = types.StringValue(ids[0])

	resp.Diagnostics.Append(r.generateUUIDs(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "created server", map[string]any{"id": ids[0]})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read is a no-op because the UUIDs only exist in state. A resource backed
// by a real API fetches the remote object here and calls
// resp.State.RemoveResource when it no longer exists.
func (r *ServerResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
}

func (r *ServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ServerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := wait(ctx, plan.UpdateDelay); err != nil {
		resp.Diagnostics.AddError("Server Update Interrupted", err.Error())

		return
	}

	// ModifyPlan leaves uuids unknown only when uuid_count changed.
	if plan.UUIDs.IsUnknown() {
		resp.Diagnostics.Append(r.generateUUIDs(ctx, &plan)...)

		if resp.Diagnostics.HasError() {
			return
		}
	}

	tflog.Info(ctx, "updated server", map[string]any{"id": plan.ID.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ServerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := wait(ctx, state.DeletionDelay); err != nil {
		resp.Diagnostics.AddError("Server Deletion Interrupted", err.Error())

		return
	}

	// The framework removes the resource from state when Delete returns
	// without error diagnostics.
	tflog.Info(ctx, "deleted server", map[string]any{"id": state.ID.ValueString()})
}

// generateUUIDs fetches uuid_count new UUIDs and stores them in the model.
func (r *ServerResource) generateUUIDs(ctx context.Context, model *ServerResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	uuids, err := r.client.Generate(ctx, int(model.UUIDCount.ValueInt64()))
	if err != nil {
		diags.AddError("Unable to Generate UUIDs", err.Error())

		return diags
	}

	model.UUIDs, diags = types.ListValueFrom(ctx, types.StringType, uuids)

	return diags
}

// wait blocks for the given number of seconds, returning early with an error
// if the operation is cancelled or times out.
func wait(ctx context.Context, seconds types.Int64) error {
	delay := time.Duration(seconds.ValueInt64()) * time.Second
	if delay <= 0 {
		return nil
	}

	tflog.Debug(ctx, "simulating a slow operation", map[string]any{"delay_seconds": seconds.ValueInt64()})

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
