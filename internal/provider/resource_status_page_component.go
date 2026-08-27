package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RealUptimeHQ/terraform-provider-realuptime/internal/client"
)

var (
	_ resource.Resource                = (*statusPageComponentResource)(nil)
	_ resource.ResourceWithConfigure   = (*statusPageComponentResource)(nil)
	_ resource.ResourceWithImportState = (*statusPageComponentResource)(nil)
)

func NewStatusPageComponentResource() resource.Resource { return &statusPageComponentResource{} }

type statusPageComponentResource struct {
	client *client.Client
}

type statusPageComponentModel struct {
	ID           types.String `tfsdk:"id"`
	StatusPageID types.String `tfsdk:"status_page_id"`
	CheckID      types.String `tfsdk:"check_id"`
	DisplayName  types.String `tfsdk:"display_name"`
	Description  types.String `tfsdk:"description"`
	SortOrder    types.Int64  `tfsdk:"sort_order"`
	GroupID      types.String `tfsdk:"group_id"`
}

func (r *statusPageComponentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page_component"
}

func (r *statusPageComponentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Publishes one of the account's checks on a status page as a public component " +
			"(`POST /status-pages/{id}/components`). The check keeps running when the component is destroyed; only its " +
			"public listing goes. A check can appear once per page (RU-5012). Agent-bound monitors are refused: they are " +
			"private by design (RU-5008). Import with `<status_page_id>/<component_id>`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Component id (UUID).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status_page_id": schema.StringAttribute{
				MarkdownDescription: "The page to publish on. Changing it replaces the component (use the dashboard's move to keep subscribers).",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"check_id": schema.StringAttribute{
				MarkdownDescription: "The check to publish. Changing it replaces the component.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Public label on the status page, 1-200 characters. Defaults to the check's name at creation and is " +
					"independent of it afterwards (renaming the check does not rename the component, or the reverse).",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.String{stringvalidator.LengthBetween(1, 200)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Tooltip text on the public page, at most 200 characters. Null renders no tooltip.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.LengthAtMost(200)},
			},
			"sort_order": schema.Int64Attribute{
				MarkdownDescription: "Position on the page (appended at creation; reordered from the dashboard).",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "Display group, if the component was grouped from the dashboard (read-only here).",
				Computed:            true,
			},
		},
	}
}

func (r *statusPageComponentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func componentToModel(c *client.StatusPageComponent, m *statusPageComponentModel) {
	m.ID = types.StringValue(c.ID)
	m.StatusPageID = types.StringValue(c.StatusPageID)
	m.CheckID = types.StringValue(c.CheckID)
	m.DisplayName = types.StringValue(c.DisplayName)
	m.Description = fromStrPtr(c.Description)
	m.SortOrder = types.Int64Value(c.SortOrder)
	m.GroupID = fromStrPtr(c.GroupID)
}

func (r *statusPageComponentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan statusPageComponentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := client.StatusPageComponentCreate{
		CheckID:     plan.CheckID.ValueString(),
		DisplayName: strPtr(plan.DisplayName),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		in.Description = client.Str(plan.Description.ValueString())
	}
	created, err := r.client.CreateStatusPageComponent(ctx, plan.StatusPageID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Creating status page component failed", err.Error())
		return
	}
	componentToModel(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *statusPageComponentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state statusPageComponentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c, err := r.client.GetStatusPageComponent(ctx, state.StatusPageID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Reading status page component failed", err.Error())
		return
	}
	componentToModel(c, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *statusPageComponentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state statusPageComponentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var in client.StatusPageComponentUpdate
	if !plan.DisplayName.IsUnknown() && !plan.DisplayName.Equal(state.DisplayName) {
		in.DisplayName = strPtr(plan.DisplayName)
	}
	if !plan.Description.Equal(state.Description) {
		in.Description = nullable(plan.Description)
	}
	updated, err := r.client.UpdateStatusPageComponent(ctx, state.StatusPageID.ValueString(), state.ID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Updating status page component failed", err.Error())
		return
	}
	componentToModel(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *statusPageComponentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state statusPageComponentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.DeleteStatusPageComponent(ctx, state.StatusPageID.ValueString(), state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting status page component failed", err.Error())
	}
}

// ImportState expects "<status_page_id>/<component_id>": the API addresses a
// component under its page, so both are needed to read it.
func (r *statusPageComponentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import id",
			fmt.Sprintf("Expected <status_page_id>/<component_id>, got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("status_page_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
