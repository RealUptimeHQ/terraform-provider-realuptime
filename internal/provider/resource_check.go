package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RealUptimeHQ/terraform-provider-realuptime/internal/client"
)

var (
	_ resource.Resource                = (*checkResource)(nil)
	_ resource.ResourceWithConfigure   = (*checkResource)(nil)
	_ resource.ResourceWithImportState = (*checkResource)(nil)
)

// liveRegions is LIVE_REGIONS from packages/db/region-catalog.ts, the
// `regions` enum in the OpenAPI document.
var liveRegions = []string{"iad", "sjc", "fra", "nrt"}

func NewCheckResource() resource.Resource { return &checkResource{} }

type checkResource struct {
	client *client.Client
}

// checkModel: attribute names are the API's own field names (request side
// camelCase -> snake_case exactly as the response already spells them).
type checkModel struct {
	ID                         types.String `tfsdk:"id"`
	AccountID                  types.String `tfsdk:"account_id"`
	Type                       types.String `tfsdk:"type"`
	Name                       types.String `tfsdk:"name"`
	URL                        types.String `tfsdk:"url"`
	IntervalSeconds            types.Int64  `tfsdk:"interval_seconds"`
	Regions                    types.Set    `tfsdk:"regions"`
	AssertionBodyOp            types.String `tfsdk:"assertion_body_op"`
	AssertionBodyValue         types.String `tfsdk:"assertion_body_value"`
	AssertionBodyCaseSensitive types.Bool   `tfsdk:"assertion_body_case_sensitive"`
	AssertionHeaderName        types.String `tfsdk:"assertion_header_name"`
	AssertionHeaderOp          types.String `tfsdk:"assertion_header_op"`
	AssertionHeaderValue       types.String `tfsdk:"assertion_header_value"`
	AssertionStatusMin         types.Int64  `tfsdk:"assertion_status_min"`
	AssertionStatusMax         types.Int64  `tfsdk:"assertion_status_max"`
	AssertionJSONPath          types.String `tfsdk:"assertion_json_path"`
	AssertionJSONOp            types.String `tfsdk:"assertion_json_op"`
	AssertionJSONValue         types.String `tfsdk:"assertion_json_value"`
}

func (r *checkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_check"
}

func (r *checkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An **http** monitor (`POST /checks`). RealUptime probes the URL from each selected region " +
			"on the interval and reports per-region truth. `name`, `url`, `interval_seconds`, `regions` and the response " +
			"assertions are all updated in place (`PATCH /checks/{id}` and `PATCH /checks/{id}/assertions`, REA-226) -- " +
			"changing any of them updates the existing check rather than replacing it, so its id and history are kept. " +
			"Other monitor types (tcp, dns, smtp, heartbeat, multistep) are created from the dashboard and can be " +
			"imported but not created here.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Check id (UUID).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"account_id": schema.StringAttribute{
				MarkdownDescription: "The owning account's id.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Monitor type. Always `http` for a check created by this resource.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name, 1-200 characters. Updated in place (`PATCH /checks/{id}`, REA-226).",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 200)},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "Public http(s) URL to probe. Private and internal addresses are refused (RU-3002). " +
					"Updated in place (`PATCH /checks/{id}`, REA-226); each new value is re-checked against the same " +
					"SSRF guard `POST /checks` uses.",
				Required:   true,
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"interval_seconds": schema.Int64Attribute{
				MarkdownDescription: "Seconds between probes from each region. Must be at or above your plan's floor " +
					"(refused with RU-3006 otherwise). Omitted, the account default applies. Updated in place " +
					"(`PATCH /checks/{id}`, REA-226).",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.Int64{int64validator.AtLeast(1)},
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"regions": schema.SetAttribute{
				MarkdownDescription: "Which live regions probe this check: any non-empty subset of `iad`, `sjc`, `fra`, `nrt`, " +
					"each at most once. Omitted, every live region probes it. Updated in place.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.OneOf(liveRegions...)),
				},
				PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			},
			"assertion_body_op": schema.StringAttribute{
				MarkdownDescription: "Body assertion: `contains` or `not_contains`. Set together with `assertion_body_value`; " +
					"the value is matched as a literal substring, never a regex.",
				Optional:   true,
				Validators: []validator.String{stringvalidator.OneOf("contains", "not_contains")},
			},
			"assertion_body_value": schema.StringAttribute{
				MarkdownDescription: "Text the response body must (or must not) contain, 1-200 characters.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 200)},
			},
			"assertion_body_case_sensitive": schema.BoolAttribute{
				MarkdownDescription: "Whether the body match is case-sensitive. Defaults to `true` when a body assertion is set; null otherwise.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{dependentBoolPlanModifier{sibling: path.Root("assertion_body_op")}},
			},
			"assertion_header_name": schema.StringAttribute{
				MarkdownDescription: "Header assertion: the response header to inspect. Set together with `assertion_header_op` and `assertion_header_value`.",
				Optional:            true,
			},
			"assertion_header_op": schema.StringAttribute{
				MarkdownDescription: "`equals` or `contains`.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.OneOf("equals", "contains")},
			},
			"assertion_header_value": schema.StringAttribute{
				MarkdownDescription: "Expected header value (or substring).",
				Optional:            true,
			},
			"assertion_status_min": schema.Int64Attribute{
				MarkdownDescription: "Status assertion: lowest acceptable HTTP status (100-599). Set together with `assertion_status_max`, min <= max.",
				Optional:            true,
				Validators:          []validator.Int64{int64validator.Between(100, 599)},
			},
			"assertion_status_max": schema.Int64Attribute{
				MarkdownDescription: "Highest acceptable HTTP status (100-599).",
				Optional:            true,
				Validators:          []validator.Int64{int64validator.Between(100, 599)},
			},
			"assertion_json_path": schema.StringAttribute{
				MarkdownDescription: "JSON assertion: a dotted path into the response body (e.g. `data.status`). Set together with `assertion_json_op`.",
				Optional:            true,
			},
			"assertion_json_op": schema.StringAttribute{
				MarkdownDescription: "`equals`, `contains`, or `exists`. `exists` takes no value.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.OneOf("equals", "contains", "exists")},
			},
			"assertion_json_value": schema.StringAttribute{
				MarkdownDescription: "Expected value at the path (omit for `exists`).",
				Optional:            true,
			},
		},
	}
}

func (r *checkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// assertionsFromModel builds the request-side assertion group from the plan.
func assertionsFromModel(m *checkModel) client.HTTPAssertions {
	return client.HTTPAssertions{
		AssertionBodyOp:            strPtr(m.AssertionBodyOp),
		AssertionBodyValue:         strPtr(m.AssertionBodyValue),
		AssertionBodyCaseSensitive: boolPtr(m.AssertionBodyCaseSensitive),
		AssertionHeaderName:        strPtr(m.AssertionHeaderName),
		AssertionHeaderOp:          strPtr(m.AssertionHeaderOp),
		AssertionHeaderValue:       strPtr(m.AssertionHeaderValue),
		AssertionStatusMin:         int64Ptr(m.AssertionStatusMin),
		AssertionStatusMax:         int64Ptr(m.AssertionStatusMax),
		AssertionJSONPath:          strPtr(m.AssertionJSONPath),
		AssertionJSONOp:            strPtr(m.AssertionJSONOp),
		AssertionJSONValue:         strPtr(m.AssertionJSONValue),
	}
}

// checkToModel writes the API entity into state.
func checkToModel(c *client.Check, m *checkModel) {
	m.ID = types.StringValue(c.ID)
	m.AccountID = types.StringValue(c.AccountID)
	m.Type = types.StringValue(c.Type)
	m.Name = types.StringValue(c.Name)
	m.URL = fromStrPtr(c.URL)
	m.IntervalSeconds = types.Int64Value(c.IntervalSeconds)
	m.Regions = stringSetValue(c.SelectedRegions)
	m.AssertionBodyOp = fromStrPtr(c.AssertionBodyOp)
	m.AssertionBodyValue = fromStrPtr(c.AssertionBodyValue)
	m.AssertionBodyCaseSensitive = fromBoolPtr(c.AssertionBodyCaseSensitive)
	m.AssertionHeaderName = fromStrPtr(c.AssertionHeaderName)
	m.AssertionHeaderOp = fromStrPtr(c.AssertionHeaderOp)
	m.AssertionHeaderValue = fromStrPtr(c.AssertionHeaderValue)
	m.AssertionStatusMin = fromInt64Ptr(c.AssertionStatusMin)
	m.AssertionStatusMax = fromInt64Ptr(c.AssertionStatusMax)
	m.AssertionJSONPath = fromStrPtr(c.AssertionJSONPath)
	m.AssertionJSONOp = fromStrPtr(c.AssertionJSONOp)
	m.AssertionJSONValue = fromStrPtr(c.AssertionJSONValue)
}

func (r *checkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan checkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	regions, diags := stringSetFrom(ctx, plan.Regions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.CheckCreate{
		Name:            plan.Name.ValueString(),
		URL:             plan.URL.ValueString(),
		IntervalSeconds: int64Ptr(plan.IntervalSeconds),
		Regions:         regions,
		HTTPAssertions:  assertionsFromModel(&plan),
	}
	created, err := r.client.CreateCheck(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Creating check failed", err.Error())
		return
	}
	checkToModel(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *checkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state checkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c, err := r.client.GetCheck(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Reading check failed", err.Error())
		return
	}
	checkToModel(c, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *checkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state checkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()

	var latest *client.Check

	// name/url/interval_seconds/regions all now go through the one PATCH
	// /checks/{id} route (REA-226 gap-list add) -- combined into a single
	// sparse CheckUpdate rather than one request per changed field, so a plan
	// that touches several of them at once is one API call, not several.
	var patch client.CheckUpdate
	patchNeeded := false
	if !plan.Name.Equal(state.Name) {
		name := plan.Name.ValueString()
		patch.Name = &name
		patchNeeded = true
	}
	if !plan.URL.Equal(state.URL) {
		url := plan.URL.ValueString()
		patch.URL = &url
		patchNeeded = true
	}
	if !plan.IntervalSeconds.IsUnknown() && !plan.IntervalSeconds.Equal(state.IntervalSeconds) {
		interval := plan.IntervalSeconds.ValueInt64()
		patch.IntervalSeconds = &interval
		patchNeeded = true
	}
	if !plan.Regions.IsUnknown() && !plan.Regions.Equal(state.Regions) {
		regions, diags := stringSetFrom(ctx, plan.Regions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		patch.Regions = regions
		patchNeeded = true
	}
	if patchNeeded {
		updated, err := r.client.UpdateCheck(ctx, id, patch)
		if err != nil {
			resp.Diagnostics.AddError("Updating check failed", err.Error())
			return
		}
		latest = updated
	}

	if assertionsChanged(&plan, &state) {
		updated, err := r.client.UpdateCheckAssertions(ctx, id, assertionsFromModel(&plan))
		if err != nil {
			resp.Diagnostics.AddError("Updating check assertions failed", err.Error())
			return
		}
		latest = updated
	}

	if latest == nil {
		// Nothing the API can change was changed; refresh so computed values stay honest.
		c, err := r.client.GetCheck(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Reading check failed", err.Error())
			return
		}
		latest = c
	}
	checkToModel(latest, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func assertionsChanged(plan, state *checkModel) bool {
	return !plan.AssertionBodyOp.Equal(state.AssertionBodyOp) ||
		!plan.AssertionBodyValue.Equal(state.AssertionBodyValue) ||
		(!plan.AssertionBodyCaseSensitive.IsUnknown() && !plan.AssertionBodyCaseSensitive.Equal(state.AssertionBodyCaseSensitive)) ||
		!plan.AssertionHeaderName.Equal(state.AssertionHeaderName) ||
		!plan.AssertionHeaderOp.Equal(state.AssertionHeaderOp) ||
		!plan.AssertionHeaderValue.Equal(state.AssertionHeaderValue) ||
		!plan.AssertionStatusMin.Equal(state.AssertionStatusMin) ||
		!plan.AssertionStatusMax.Equal(state.AssertionStatusMax) ||
		!plan.AssertionJSONPath.Equal(state.AssertionJSONPath) ||
		!plan.AssertionJSONOp.Equal(state.AssertionJSONOp) ||
		!plan.AssertionJSONValue.Equal(state.AssertionJSONValue)
}

func (r *checkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state checkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteCheck(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting check failed", err.Error())
	}
}

func (r *checkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid import id", fmt.Sprintf("Expected a check id (UUID), got %q.", req.ID))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
