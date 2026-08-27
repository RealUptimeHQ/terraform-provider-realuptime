package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/RealUptimeHQ/terraform-provider-realuptime/internal/client"
)

var (
	_ datasource.DataSource                     = (*statusPageDataSource)(nil)
	_ datasource.DataSourceWithConfigure        = (*statusPageDataSource)(nil)
	_ datasource.DataSourceWithConfigValidators = (*statusPageDataSource)(nil)
)

func NewStatusPageDataSource() datasource.DataSource { return &statusPageDataSource{} }

type statusPageDataSource struct {
	client *client.Client
}

func (d *statusPageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page"
}

func (d *statusPageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up one of the account's status pages by `id` or by `slug` (`GET /status-pages/{id}`, " +
			"or `GET /status-pages` filtered). Typical use: find the page created at signup to publish components on it " +
			"without importing it.",
		Attributes: map[string]schema.Attribute{
			"id":               schema.StringAttribute{MarkdownDescription: "Status page id (UUID). Exactly one of `id` or `slug` is required.", Optional: true, Computed: true},
			"slug":             schema.StringAttribute{MarkdownDescription: "Public slug. Exactly one of `id` or `slug` is required.", Optional: true, Computed: true},
			"account_id":       schema.StringAttribute{MarkdownDescription: "The owning account's id.", Computed: true},
			"name":             schema.StringAttribute{MarkdownDescription: "Page name.", Computed: true},
			"page_title":       schema.StringAttribute{MarkdownDescription: "Heading shown instead of `name`, if set.", Computed: true},
			"page_description": schema.StringAttribute{MarkdownDescription: "Subtitle under the heading, if set.", Computed: true},
			"logo_url":         schema.StringAttribute{MarkdownDescription: "Logo URL, if set.", Computed: true},
			"hide_branding":    schema.BoolAttribute{MarkdownDescription: "Whether the attribution is hidden (honoured on paid plans only).", Computed: true},
			"accent_color":     schema.StringAttribute{MarkdownDescription: "Chrome accent `#rrggbb`, if set.", Computed: true},
			"homepage_url":     schema.StringAttribute{MarkdownDescription: "Logo link, if set.", Computed: true},
			"support_url":      schema.StringAttribute{MarkdownDescription: "Support link, if set.", Computed: true},
			"theme_preset":     schema.StringAttribute{MarkdownDescription: "Theme preset id, if set.", Computed: true},
			"font_choice":      schema.StringAttribute{MarkdownDescription: "Font choice id, if set.", Computed: true},
			"locale":           schema.StringAttribute{MarkdownDescription: "Formatting locale, if set (null is en-US).", Computed: true},
			"custom_domain":    schema.StringAttribute{MarkdownDescription: "Custom domain, if attached.", Computed: true},
			"domain_status":    schema.StringAttribute{MarkdownDescription: "Custom-domain lifecycle: none, pending_dns, issuing, issued, error.", Computed: true},
		},
	}
}

func (d *statusPageDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("slug")),
	}
}

func (d *statusPageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *statusPageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg statusPageModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var page *client.StatusPage
	if !cfg.ID.IsNull() && !cfg.ID.IsUnknown() {
		p, err := d.client.GetStatusPage(ctx, cfg.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Reading status page failed", err.Error())
			return
		}
		page = p
	} else {
		pages, err := d.client.ListStatusPages(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Listing status pages failed", err.Error())
			return
		}
		slug := cfg.Slug.ValueString()
		for i := range pages {
			if pages[i].Slug == slug {
				page = &pages[i]
				break
			}
		}
		if page == nil {
			resp.Diagnostics.AddError("Status page not found", fmt.Sprintf("No status page on this account has slug %q.", slug))
			return
		}
	}

	var out statusPageModel
	statusPageToModel(page, &out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
