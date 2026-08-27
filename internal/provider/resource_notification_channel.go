package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RealUptimeHQ/terraform-provider-realuptime/internal/client"
)

var (
	_ resource.Resource                = (*notificationChannelResource)(nil)
	_ resource.ResourceWithConfigure   = (*notificationChannelResource)(nil)
	_ resource.ResourceWithImportState = (*notificationChannelResource)(nil)
)

// NOTIFICATION_CHANNEL_TYPES from packages/db/notification-formatters.ts.
// sms/voice/mobile_push are accepted by the schema but not yet deliverable
// (provider_configured is false); the API refuses them today.
var notificationChannelTypes = []string{"discord", "msteams", "opsgenie", "generic_json", "sms", "voice", "mobile_push"}

func NewNotificationChannelResource() resource.Resource { return &notificationChannelResource{} }

type notificationChannelResource struct {
	client *client.Client
}

type notificationChannelModel struct {
	ID                 types.String `tfsdk:"id"`
	AccountID          types.String `tfsdk:"account_id"`
	Type               types.String `tfsdk:"type"`
	Name               types.String `tfsdk:"name"`
	Config             types.Map    `tfsdk:"config"`
	Enabled            types.Bool   `tfsdk:"enabled"`
	ConfigSummary      types.String `tfsdk:"config_summary"`
	VerifiedAt         types.String `tfsdk:"verified_at"`
	ProviderConfigured types.Bool   `tfsdk:"provider_configured"`
	TypeLabel          types.String `tfsdk:"type_label"`
}

func (r *notificationChannelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_channel"
}

func (r *notificationChannelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A notification channel in the account's registry (`POST /notification-channels`): a Discord " +
			"webhook, a Microsoft Teams webhook, an Opsgenie integration, or a generic JSON endpoint (Zapier and the like). " +
			"Attach it to alert rules from the dashboard. Email, Slack, plain webhooks and PagerDuty are configured on " +
			"alert rules directly, not here. The credential in `config` is WRITE-ONLY: the API never returns it, so " +
			"Terraform keeps the value you set and compares against that; `config_summary` is the redacted form the API " +
			"does return. A new channel is unverified until a test send succeeds from the dashboard.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Channel id (UUID).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"account_id": schema.StringAttribute{
				MarkdownDescription: "The owning account's id.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "One of `discord`, `msteams`, `opsgenie`, `generic_json` (`sms`, `voice`, `mobile_push` are reserved and refused today). Immutable: changing it replaces the channel.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf(notificationChannelTypes...)},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name, 1-120 characters, unique per account (RU-3009).",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 120)},
			},
			"config": schema.MapAttribute{
				MarkdownDescription: "Per-type settings, all string values. `discord` and `msteams`: `webhookUrl` (an https " +
					"webhook URL on the vendor's own host). `generic_json`: `url` (https, public address). `opsgenie`: `apiKey` " +
					"and `region` (`us` or `eu`). Replacing the config clears verification and the last test result.",
				ElementType: types.StringType,
				Required:    true,
				Sensitive:   true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "False soft-disables the channel: it delivers nothing but keeps its alert-rule attachments, so re-enabling restores them. Default true.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"config_summary": schema.StringAttribute{
				MarkdownDescription: "Redacted, human-readable description of the stored config (a URL origin, \"EU region, API key stored\"). Never the credential.",
				Computed:            true,
			},
			"verified_at": schema.StringAttribute{
				MarkdownDescription: "When a test send last succeeded against the CURRENT config; null until one has.",
				Computed:            true,
			},
			"provider_configured": schema.BoolAttribute{
				MarkdownDescription: "Whether this build can actually deliver to the type (false for the reserved sms/voice/mobile_push stubs).",
				Computed:            true,
			},
			"type_label": schema.StringAttribute{
				MarkdownDescription: "The type's display label (e.g. \"Microsoft Teams\").",
				Computed:            true,
			},
		},
	}
}

func (r *notificationChannelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// channelToModel writes the API entity into state. `config` is deliberately
// NOT touched: the API never returns it, so the planned/stored value stands.
func channelToModel(c *client.NotificationChannel, m *notificationChannelModel) {
	m.ID = types.StringValue(c.ID)
	m.AccountID = types.StringValue(c.AccountID)
	m.Type = types.StringValue(c.Type)
	m.Name = types.StringValue(c.Name)
	m.Enabled = types.BoolValue(c.DisabledAt == nil)
	m.ConfigSummary = types.StringValue(c.ConfigSummary)
	m.VerifiedAt = fromStrPtr(c.VerifiedAt)
	m.ProviderConfigured = types.BoolValue(c.ProviderConfigured)
	m.TypeLabel = types.StringValue(c.TypeLabel)
}

func configFrom(ctx context.Context, v types.Map) (map[string]string, bool) {
	if v.IsNull() || v.IsUnknown() {
		return nil, false
	}
	out := map[string]string{}
	if diags := v.ElementsAs(ctx, &out, false); diags.HasError() {
		return nil, false
	}
	return out, true
}

func (r *notificationChannelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan notificationChannelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, _ := configFrom(ctx, plan.Config)
	created, err := r.client.CreateNotificationChannel(ctx, client.NotificationChannelCreate{
		Type:   plan.Type.ValueString(),
		Name:   plan.Name.ValueString(),
		Config: cfg,
	})
	if err != nil {
		resp.Diagnostics.AddError("Creating notification channel failed", err.Error())
		return
	}
	if !plan.Enabled.IsUnknown() && !plan.Enabled.ValueBool() {
		f := false
		created, err = r.client.UpdateNotificationChannel(ctx, created.ID, client.NotificationChannelUpdate{Enabled: &f})
		if err != nil {
			resp.Diagnostics.AddError("Disabling notification channel after create failed", err.Error())
			return
		}
	}
	channelToModel(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *notificationChannelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state notificationChannelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c, err := r.client.GetNotificationChannel(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Reading notification channel failed", err.Error())
		return
	}
	channelToModel(c, &state)
	if state.Config.IsNull() {
		// Imported: the credential is not readable, so leave an empty map and
		// let the next plan show the config being (re)set.
		state.Config = types.MapValueMust(types.StringType, nil)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *notificationChannelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state notificationChannelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var in client.NotificationChannelUpdate
	if !plan.Name.Equal(state.Name) {
		in.Name = strPtr(plan.Name)
	}
	if !plan.Config.Equal(state.Config) {
		in.Config, _ = configFrom(ctx, plan.Config)
	}
	if !plan.Enabled.IsUnknown() && !plan.Enabled.Equal(state.Enabled) {
		in.Enabled = boolPtr(plan.Enabled)
	}
	updated, err := r.client.UpdateNotificationChannel(ctx, state.ID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Updating notification channel failed", err.Error())
		return
	}
	channelToModel(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *notificationChannelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state notificationChannelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteNotificationChannel(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting notification channel failed", err.Error())
	}
}

func (r *notificationChannelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
