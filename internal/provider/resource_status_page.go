package provider

import (
	"context"
	"regexp"

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
	_ resource.Resource                = (*statusPageResource)(nil)
	_ resource.ResourceWithConfigure   = (*statusPageResource)(nil)
	_ resource.ResourceWithImportState = (*statusPageResource)(nil)
)

// Enums from statusPageUpdateShape in packages/db/api-schemas.ts.
var (
	statusPageThemePresets = []string{"signal", "midnight", "slate", "ocean", "plum", "graphite"}
	statusPageFontChoices  = []string{"inter", "system", "space-grotesk"}
	statusPageLocales      = []string{"en-US", "en-GB", "de-DE", "fr-FR", "es-ES", "it-IT", "pt-BR", "nl-NL", "sv-SE", "ja-JP", "ko-KR", "zh-CN"}
	statusPageSlugPattern  = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

func NewStatusPageResource() resource.Resource { return &statusPageResource{} }

type statusPageResource struct {
	client *client.Client
}

type statusPageModel struct {
	ID              types.String `tfsdk:"id"`
	AccountID       types.String `tfsdk:"account_id"`
	Name            types.String `tfsdk:"name"`
	Slug            types.String `tfsdk:"slug"`
	PageTitle       types.String `tfsdk:"page_title"`
	PageDescription types.String `tfsdk:"page_description"`
	LogoURL         types.String `tfsdk:"logo_url"`
	HideBranding    types.Bool   `tfsdk:"hide_branding"`
	AccentColor     types.String `tfsdk:"accent_color"`
	HomepageURL     types.String `tfsdk:"homepage_url"`
	SupportURL      types.String `tfsdk:"support_url"`
	ThemePreset     types.String `tfsdk:"theme_preset"`
	FontChoice      types.String `tfsdk:"font_choice"`
	Locale          types.String `tfsdk:"locale"`
	CustomDomain    types.String `tfsdk:"custom_domain"`
	DomainStatus    types.String `tfsdk:"domain_status"`
}

func (r *statusPageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page"
}

func (r *statusPageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A public status page (`POST /status-pages`, `PATCH /status-pages/{id}`). Every account " +
			"gets one page at signup; this resource creates ADDITIONAL pages (Free includes one page, Pro and " +
			"Business unlimited: RU-2006 otherwise). Import the signup page by id to manage it here. Deleting a page " +
			"removes everything published on it (components, incidents, maintenance windows, subscribers); the " +
			"account's last page cannot be deleted (RU-5011). Custom domains and passwords are managed from the dashboard.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Status page id (UUID).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"account_id": schema.StringAttribute{
				MarkdownDescription: "The owning account's id.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Page name, 1-200 characters. Shown as the heading unless `page_title` is set.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 200)},
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "Public slug: lowercase letters, digits and single hyphens, at most 63 characters " +
					"(it is also the page's subdomain label). One global namespace across every account (RU-5010 if " +
					"taken); reserved words are refused. Omitted, a neutral `status-xxxxxx` slug is generated. Renaming " +
					"keeps the old slug redirecting.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 63),
					stringvalidator.RegexMatches(statusPageSlugPattern, "lowercase letters, numbers and hyphens only; no leading/trailing hyphen"),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"page_title": schema.StringAttribute{
				MarkdownDescription: "Heading shown instead of `name`, at most 200 characters. Null shows `name`.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.LengthAtMost(200)},
			},
			"page_description": schema.StringAttribute{
				MarkdownDescription: "Subtitle under the heading, at most 280 characters.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.LengthAtMost(280)},
			},
			"logo_url": schema.StringAttribute{
				MarkdownDescription: "https URL of a PNG/JPEG/SVG/WebP logo, served through RealUptime's proxy. Must resolve to a public address.",
				Optional:            true,
			},
			"hide_branding": schema.BoolAttribute{
				MarkdownDescription: "Hide the \"Powered by realuptime\" attribution. Stored as asked; honoured only on paid plans (a Free page keeps the badge).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"accent_color": schema.StringAttribute{
				MarkdownDescription: "Chrome accent as `#rrggbb` (lowercase). Colors links and the header rule only; the operational/degraded/down status colors are never customer-configurable. Accents too light to read are refused.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`^#[0-9a-f]{6}$`), "must be #rrggbb, lowercase")},
			},
			"homepage_url": schema.StringAttribute{
				MarkdownDescription: "https URL the logo links to.",
				Optional:            true,
			},
			"support_url": schema.StringAttribute{
				MarkdownDescription: "https URL for the page's Support link.",
				Optional:            true,
			},
			"theme_preset": schema.StringAttribute{
				MarkdownDescription: "Curated palette and density: one of `signal`, `midnight`, `slate`, `ocean`, `plum`, `graphite`. Null uses `accent_color` at comfortable density.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.OneOf(statusPageThemePresets...)},
			},
			"font_choice": schema.StringAttribute{
				MarkdownDescription: "One of `inter`, `system`, `space-grotesk`. Null is the default Inter body / Space Grotesk heading pairing.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.OneOf(statusPageFontChoices...)},
			},
			"locale": schema.StringAttribute{
				MarkdownDescription: "BCP-47 tag for date/time/number formatting on the public page (never a translation): one of " +
					"`en-US`, `en-GB`, `de-DE`, `fr-FR`, `es-ES`, `it-IT`, `pt-BR`, `nl-NL`, `sv-SE`, `ja-JP`, `ko-KR`, `zh-CN`. Null is `en-US`.",
				Optional:   true,
				Validators: []validator.String{stringvalidator.OneOf(statusPageLocales...)},
			},
			"custom_domain": schema.StringAttribute{
				MarkdownDescription: "Custom domain, if one is attached from the dashboard (read-only here).",
				Computed:            true,
			},
			"domain_status": schema.StringAttribute{
				MarkdownDescription: "Custom-domain lifecycle: `none`, `pending_dns`, `issuing`, `issued`, `error` (read-only here).",
				Computed:            true,
			},
		},
	}
}

func (r *statusPageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func statusPageToModel(p *client.StatusPage, m *statusPageModel) {
	m.ID = types.StringValue(p.ID)
	m.AccountID = types.StringValue(p.AccountID)
	m.Name = types.StringValue(p.Name)
	m.Slug = types.StringValue(p.Slug)
	m.PageTitle = fromStrPtr(p.PageTitle)
	m.PageDescription = fromStrPtr(p.PageDescription)
	m.LogoURL = fromStrPtr(p.LogoURL)
	m.HideBranding = types.BoolValue(p.HideBranding)
	m.AccentColor = fromStrPtr(p.AccentColor)
	m.HomepageURL = fromStrPtr(p.HomepageURL)
	m.SupportURL = fromStrPtr(p.SupportURL)
	m.ThemePreset = fromStrPtr(p.ThemePreset)
	m.FontChoice = fromStrPtr(p.FontChoice)
	m.Locale = fromStrPtr(p.Locale)
	m.CustomDomain = fromStrPtr(p.CustomDomain)
	m.DomainStatus = types.StringValue(p.DomainStatus)
}

// brandingUpdate builds the PATCH body. On create (prior is the empty model)
// every non-null optional is sent so the page leaves Create matching the plan;
// on update only fields that differ from prior are sent, null meaning "clear".
func brandingUpdate(plan, prior *statusPageModel, create bool) (client.StatusPageUpdate, bool) {
	var u client.StatusPageUpdate
	changed := false
	str := func(p, s types.String, set func(client.NullableString)) {
		if create {
			if !p.IsNull() && !p.IsUnknown() {
				set(nullable(p))
				changed = true
			}
			return
		}
		if !p.IsUnknown() && !p.Equal(s) {
			set(nullable(p))
			changed = true
		}
	}
	if !create && !plan.Name.Equal(prior.Name) {
		u.Name = strPtr(plan.Name)
		changed = true
	}
	if !create && !plan.Slug.IsUnknown() && !plan.Slug.Equal(prior.Slug) {
		u.Slug = strPtr(plan.Slug)
		changed = true
	}
	str(plan.PageTitle, prior.PageTitle, func(v client.NullableString) { u.PageTitle = v })
	str(plan.PageDescription, prior.PageDescription, func(v client.NullableString) { u.PageDescription = v })
	str(plan.LogoURL, prior.LogoURL, func(v client.NullableString) { u.LogoURL = v })
	str(plan.AccentColor, prior.AccentColor, func(v client.NullableString) { u.AccentColor = v })
	str(plan.HomepageURL, prior.HomepageURL, func(v client.NullableString) { u.HomepageURL = v })
	str(plan.SupportURL, prior.SupportURL, func(v client.NullableString) { u.SupportURL = v })
	str(plan.ThemePreset, prior.ThemePreset, func(v client.NullableString) { u.ThemePreset = v })
	str(plan.FontChoice, prior.FontChoice, func(v client.NullableString) { u.FontChoice = v })
	str(plan.Locale, prior.Locale, func(v client.NullableString) { u.Locale = v })
	if !plan.HideBranding.IsUnknown() {
		if (create && plan.HideBranding.ValueBool()) || (!create && !plan.HideBranding.Equal(prior.HideBranding)) {
			u.HideBranding = boolPtr(plan.HideBranding)
			changed = true
		}
	}
	return u, changed
}

func (r *statusPageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan statusPageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.CreateStatusPage(ctx, client.StatusPageCreate{
		Name: plan.Name.ValueString(),
		Slug: strPtr(plan.Slug),
	})
	if err != nil {
		resp.Diagnostics.AddError("Creating status page failed", err.Error())
		return
	}
	// Branding is a second call: POST /status-pages takes only name and slug.
	if u, changed := brandingUpdate(&plan, &statusPageModel{}, true); changed {
		updated, err := r.client.UpdateStatusPage(ctx, created.ID, u)
		if err != nil {
			// The page exists; record it so a retry updates rather than duplicates.
			statusPageToModel(created, &plan)
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
			resp.Diagnostics.AddError("Applying status page branding failed", err.Error())
			return
		}
		created = updated
	}
	statusPageToModel(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *statusPageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state statusPageModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p, err := r.client.GetStatusPage(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Reading status page failed", err.Error())
		return
	}
	statusPageToModel(p, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *statusPageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state statusPageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	u, changed := brandingUpdate(&plan, &state, false)
	var latest *client.StatusPage
	var err error
	if changed {
		latest, err = r.client.UpdateStatusPage(ctx, state.ID.ValueString(), u)
		if err != nil {
			resp.Diagnostics.AddError("Updating status page failed", err.Error())
			return
		}
	} else {
		latest, err = r.client.GetStatusPage(ctx, state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Reading status page failed", err.Error())
			return
		}
	}
	statusPageToModel(latest, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *statusPageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state statusPageModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteStatusPage(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting status page failed", err.Error())
	}
}

func (r *statusPageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
