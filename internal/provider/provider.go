// Package provider implements the `realuptime` Terraform provider on
// terraform-plugin-framework. Every resource is a thin mapping from the
// REST API v1 (https://realuptime.io/api/v1/openapi.json): attribute names
// are the API's request field names (snake_cased the Terraform way, which
// is also how the API returns them), and every description below is
// adapted from the OpenAPI document's own field descriptions.
package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RealUptimeHQ/terraform-provider-realuptime/internal/client"
)

var _ provider.Provider = (*realuptimeProvider)(nil)

type realuptimeProvider struct {
	version string
}

// New returns the provider constructor providerserver and the acceptance
// tests both use.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &realuptimeProvider{version: version}
	}
}

type providerModel struct {
	APIKey  types.String `tfsdk:"api_key"`
	BaseURL types.String `tfsdk:"base_url"`
}

func (p *realuptimeProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "realuptime"
	resp.Version = p.version
}

func (p *realuptimeProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage RealUptime monitors (checks), status pages, status page components and " +
			"notification channels through the REST API v1. The REST API is available on the Growth and Scale " +
			"plans; generate an API key with **read_write** permission from the dashboard's API & MCP access " +
			"section (a read-only key can only read).",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "API key, sent as `Authorization: Bearer <key>` on every request. " +
					"Falls back to the `REALUPTIME_API_KEY` environment variable. Keep it out of your configuration: " +
					"prefer the environment variable or a secret store.",
				Optional:  true,
				Sensitive: true,
			},
			"base_url": schema.StringAttribute{
				MarkdownDescription: "API root. Defaults to `" + client.DefaultBaseURL + "`, or the `REALUPTIME_API_URL` " +
					"environment variable when set (the same variable the RealUptime CLI reads). Override it only to " +
					"point at a local development server.",
				Optional: true,
			},
		},
	}
}

func (p *realuptimeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if cfg.APIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("api_key"), "Unknown API key",
			"The provider cannot be configured with an unknown api_key. Set it to a known value or use REALUPTIME_API_KEY.")
		return
	}

	apiKey := os.Getenv("REALUPTIME_API_KEY")
	if !cfg.APIKey.IsNull() {
		apiKey = cfg.APIKey.ValueString()
	}
	baseURL := os.Getenv("REALUPTIME_API_URL")
	if !cfg.BaseURL.IsNull() && !cfg.BaseURL.IsUnknown() {
		baseURL = cfg.BaseURL.ValueString()
	}
	if baseURL == "" {
		baseURL = client.DefaultBaseURL
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(path.Root("api_key"), "Missing API key",
			"Set api_key in the provider block or export REALUPTIME_API_KEY. Generate a read_write key from the "+
				"RealUptime dashboard's API & MCP access section (Growth or Scale plan).")
		return
	}

	c := client.New(baseURL, apiKey)
	client.UserAgent = "terraform-provider-realuptime/" + p.version
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *realuptimeProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCheckResource,
		NewStatusPageResource,
		NewStatusPageComponentResource,
		NewNotificationChannelResource,
	}
}

func (p *realuptimeProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewStatusPageDataSource,
	}
}

// clientFromProviderData is the shared Configure body for every resource and
// data source: ProviderData is nil during validation passes, so a nil is not
// an error here.
func clientFromProviderData(data any, diags interface {
	AddError(summary, detail string)
}) *client.Client {
	if data == nil {
		return nil
	}
	c, ok := data.(*client.Client)
	if !ok {
		diags.AddError("Unexpected provider data", "Expected *client.Client; this is a bug in the provider.")
		return nil
	}
	return c
}
