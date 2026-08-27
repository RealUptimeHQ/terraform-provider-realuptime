package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RealUptimeHQ/terraform-provider-realuptime/internal/client"
)

// Unit tests that need neither Terraform nor the API: every schema is
// internally consistent (the framework validates attribute flags and plan
// modifiers in ValidateImplementation), and the entity -> state mappings
// carry nulls faithfully.

func TestProviderSchemaIsValid(t *testing.T) {
	p := New("test")()
	var resp provider.SchemaResponse
	p.Schema(context.Background(), provider.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatal(diags)
	}
	if _, ok := resp.Schema.Attributes["api_key"]; !ok {
		t.Fatal("api_key missing")
	}
	if !resp.Schema.Attributes["api_key"].IsSensitive() {
		t.Fatal("api_key must be sensitive")
	}
}

func TestEveryResourceSchemaIsValid(t *testing.T) {
	p := New("test")()
	for _, ctor := range p.Resources(context.Background()) {
		r := ctor()
		var meta resource.MetadataResponse
		r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "realuptime"}, &meta)
		var resp resource.SchemaResponse
		r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("%s: %v", meta.TypeName, resp.Diagnostics)
		}
		if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
			t.Fatalf("%s: %v", meta.TypeName, diags)
		}
		if _, ok := resp.Schema.Attributes["id"]; !ok {
			t.Fatalf("%s has no id attribute", meta.TypeName)
		}
		for name, attr := range resp.Schema.Attributes {
			if attr.GetMarkdownDescription() == "" {
				t.Fatalf("%s.%s has no description (docs are generated from these)", meta.TypeName, name)
			}
		}
	}
	wantTypes := map[string]bool{
		"realuptime_check": false, "realuptime_status_page": false,
		"realuptime_status_page_component": false, "realuptime_notification_channel": false,
	}
	for _, ctor := range p.Resources(context.Background()) {
		var meta resource.MetadataResponse
		ctor().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "realuptime"}, &meta)
		if _, ok := wantTypes[meta.TypeName]; !ok {
			t.Fatalf("unexpected resource %s", meta.TypeName)
		}
		wantTypes[meta.TypeName] = true
	}
	for name, seen := range wantTypes {
		if !seen {
			t.Fatalf("resource %s not registered", name)
		}
	}
}

func TestDataSourceSchemaIsValid(t *testing.T) {
	p := New("test")()
	sources := p.DataSources(context.Background())
	if len(sources) != 1 {
		t.Fatalf("expected 1 data source, got %d", len(sources))
	}
	ds := sources[0]()
	var meta datasource.MetadataResponse
	ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "realuptime"}, &meta)
	if meta.TypeName != "realuptime_status_page" {
		t.Fatalf("data source type %s", meta.TypeName)
	}
	var resp datasource.SchemaResponse
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatal(diags)
	}
}

func TestCheckToModelCarriesNulls(t *testing.T) {
	url := "https://e.x"
	op := "contains"
	cs := true
	var m checkModel
	checkToModel(&client.Check{
		ID: "c1", AccountID: "a", Type: "http", Name: "API", URL: &url, IntervalSeconds: 60,
		SelectedRegions: []string{"iad", "fra"}, AssertionBodyOp: &op, AssertionBodyCaseSensitive: &cs,
	}, &m)
	if m.ID.ValueString() != "c1" || m.URL.ValueString() != "https://e.x" || m.IntervalSeconds.ValueInt64() != 60 {
		t.Fatalf("basic fields %+v", m)
	}
	if m.AssertionBodyOp.ValueString() != "contains" || !m.AssertionBodyCaseSensitive.ValueBool() {
		t.Fatalf("assertions %+v", m)
	}
	if !m.AssertionHeaderName.IsNull() || !m.AssertionStatusMin.IsNull() || !m.AssertionJSONOp.IsNull() {
		t.Fatalf("absent assertions must be null: %+v", m)
	}
	if m.Regions.IsNull() || len(m.Regions.Elements()) != 2 {
		t.Fatalf("regions %+v", m.Regions)
	}
	if m.Type.ValueString() != "http" {
		t.Fatalf("type %v", m.Type)
	}
}

func TestAssertionsFromModelOmitsNulls(t *testing.T) {
	m := checkModel{
		AssertionStatusMin: types.Int64Value(200),
		AssertionStatusMax: types.Int64Value(299),
		AssertionBodyOp:    types.StringNull(),
	}
	a := assertionsFromModel(&m)
	if a.AssertionStatusMin == nil || *a.AssertionStatusMin != 200 || a.AssertionBodyOp != nil || a.AssertionJSONPath != nil {
		t.Fatalf("%+v", a)
	}
}

func TestAssertionsChangedIgnoresUnknownCaseSensitive(t *testing.T) {
	state := checkModel{AssertionBodyOp: types.StringValue("contains"), AssertionBodyValue: types.StringValue("ok"), AssertionBodyCaseSensitive: types.BoolValue(true)}
	plan := state
	plan.AssertionBodyCaseSensitive = types.BoolUnknown()
	if assertionsChanged(&plan, &state) {
		t.Fatal("an unknown computed case-sensitivity alone is not a change")
	}
	plan.AssertionBodyValue = types.StringValue("OK")
	if !assertionsChanged(&plan, &state) {
		t.Fatal("a changed value is a change")
	}
}

func TestStatusPageToModelAndBrandingUpdate(t *testing.T) {
	title := "Acme Status"
	var m statusPageModel
	statusPageToModel(&client.StatusPage{ID: "p1", AccountID: "a", Slug: "acme", Name: "Acme", PageTitle: &title, DomainStatus: "none", HideBranding: true}, &m)
	if m.PageTitle.ValueString() != "Acme Status" || !m.HideBranding.ValueBool() || !m.LogoURL.IsNull() || !m.CustomDomain.IsNull() {
		t.Fatalf("%+v", m)
	}

	// Create path: only set optionals go, name/slug do not (POST already set them).
	plan := statusPageModel{Name: types.StringValue("Acme"), Slug: types.StringValue("acme"), PageTitle: types.StringValue("T"), HideBranding: types.BoolValue(true), Locale: types.StringNull()}
	u, changed := brandingUpdate(&plan, &statusPageModel{}, true)
	if !changed || u.Name != nil || u.Slug != nil || !u.PageTitle.Set || *u.PageTitle.Value != "T" || u.Locale.Set || u.HideBranding == nil || !*u.HideBranding {
		t.Fatalf("create update %+v", u)
	}
	// Nothing optional set on create: no PATCH at all.
	if _, changed := brandingUpdate(&statusPageModel{Name: types.StringValue("A"), HideBranding: types.BoolValue(false)}, &statusPageModel{}, true); changed {
		t.Fatal("bare create must not PATCH")
	}

	// Update path: a cleared title is sent as explicit null; an untouched field is omitted.
	prior := statusPageModel{Name: types.StringValue("Acme"), Slug: types.StringValue("acme"), PageTitle: types.StringValue("T"), HideBranding: types.BoolValue(true), Locale: types.StringValue("de-DE")}
	next := prior
	next.PageTitle = types.StringNull()
	next.Name = types.StringValue("Acme Inc")
	u, changed = brandingUpdate(&next, &prior, false)
	if !changed || u.Name == nil || *u.Name != "Acme Inc" || !u.PageTitle.Set || u.PageTitle.Value != nil || u.Locale.Set || u.HideBranding != nil || u.Slug != nil {
		t.Fatalf("update %+v", u)
	}
}

func TestChannelToModelLeavesConfigAlone(t *testing.T) {
	m := notificationChannelModel{Config: types.MapValueMust(types.StringType, map[string]attr.Value{
		"webhookUrl": types.StringValue("https://discord.com/api/webhooks/1/x"),
	})}
	disabled := "2026-08-23T00:00:00Z"
	channelToModel(&client.NotificationChannel{ID: "n1", Type: "discord", Name: "Ops", ConfigSummary: "discord.com/...", DisabledAt: &disabled, ProviderConfigured: true, TypeLabel: "Discord"}, &m)
	if m.Enabled.ValueBool() {
		t.Fatal("disabled_at set must read as enabled=false")
	}
	if m.ConfigSummary.ValueString() != "discord.com/..." || !m.VerifiedAt.IsNull() {
		t.Fatalf("%+v", m)
	}
	if len(m.Config.Elements()) != 1 {
		t.Fatal("config must be left as planned; the API never returns it")
	}
}

func TestComponentToModel(t *testing.T) {
	var m statusPageComponentModel
	componentToModel(&client.StatusPageComponent{ID: "k1", StatusPageID: "p1", CheckID: "c1", DisplayName: "API", SortOrder: 2}, &m)
	if m.ID.ValueString() != "k1" || m.StatusPageID.ValueString() != "p1" || m.SortOrder.ValueInt64() != 2 || !m.Description.IsNull() || !m.GroupID.IsNull() {
		t.Fatalf("%+v", m)
	}
}
