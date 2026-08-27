package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// Each resource's CRUD mapping: the request bodies use the API's camelCase
// request field names exactly as packages/db/api-schemas.ts spells them, and
// the responses decode from the snake_case entity shapes.

func TestStatusPageCRUDMapping(t *testing.T) {
	c, seen := fakeAPI(t, map[string]func(http.ResponseWriter, *http.Request){
		"POST /api/v1/status-pages":      jsonResponse(201, `{"statusPage":{"id":"p1","account_id":"a","slug":"acme","name":"Acme","custom_domain":null,"page_title":null,"page_description":null,"logo_url":null,"domain_status":"none","domain_error":null,"hide_branding":false,"accent_color":null,"homepage_url":null,"support_url":null,"theme_preset":null,"font_choice":null,"locale":null}}`),
		"GET /api/v1/status-pages/p1":    jsonResponse(200, `{"statusPage":{"id":"p1","slug":"acme","name":"Acme","domain_status":"none","hide_branding":true,"accent_color":"#112233"}}`),
		"PATCH /api/v1/status-pages/p1":  jsonResponse(200, `{"statusPage":{"id":"p1","slug":"acme-2","name":"Acme","domain_status":"none","page_title":null}}`),
		"DELETE /api/v1/status-pages/p1": func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) },
	})
	ctx := context.Background()

	slug := "acme"
	p, err := c.CreateStatusPage(ctx, StatusPageCreate{Name: "Acme", Slug: &slug})
	if err != nil || p.ID != "p1" || p.Slug != "acme" {
		t.Fatalf("create: %+v %v", p, err)
	}
	if b := (*seen)[0].Body; b["name"] != "Acme" || b["slug"] != "acme" {
		t.Fatalf("create body %v", b)
	}

	got, err := c.GetStatusPage(ctx, "p1")
	if err != nil || !got.HideBranding || got.AccentColor == nil || *got.AccentColor != "#112233" {
		t.Fatalf("get: %+v %v", got, err)
	}

	newSlug := "acme-2"
	u := StatusPageUpdate{Slug: &newSlug, PageTitle: Null(), AccentColor: Str("#abcdef")}
	upd, err := c.UpdateStatusPage(ctx, "p1", u)
	if err != nil || upd.Slug != "acme-2" {
		t.Fatalf("update: %+v %v", upd, err)
	}
	body := (*seen)[2].Body
	if body["slug"] != "acme-2" {
		t.Fatalf("update body slug %v", body)
	}
	if v, present := body["pageTitle"]; !present || v != nil {
		t.Fatalf("explicit null must be sent as null: %v", body)
	}
	if body["accentColor"] != "#abcdef" {
		t.Fatalf("accentColor %v", body)
	}
	if _, present := body["pageDescription"]; present {
		t.Fatalf("unset nullable must be omitted: %v", body)
	}
	if _, present := body["name"]; present {
		t.Fatalf("unset name must be omitted: %v", body)
	}

	if err := c.DeleteStatusPage(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
}

func TestNullableStringJSON(t *testing.T) {
	type wrap struct {
		A NullableString `json:"a,omitzero"`
		B NullableString `json:"b,omitzero"`
		C NullableString `json:"c,omitzero"`
	}
	raw, err := json.Marshal(wrap{B: Null(), C: Str("x")})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"b":null,"c":"x"}` {
		t.Fatalf("got %s", raw)
	}
}

func TestStatusPageComponentCRUDMapping(t *testing.T) {
	c, seen := fakeAPI(t, map[string]func(http.ResponseWriter, *http.Request){
		"POST /api/v1/status-pages/p1/components":        jsonResponse(201, `{"component":{"id":"k1","status_page_id":"p1","check_id":"c1","display_name":"API","description":null,"sort_order":3,"group_id":null}}`),
		"GET /api/v1/status-pages/p1/components/k1":      jsonResponse(200, `{"component":{"id":"k1","status_page_id":"p1","check_id":"c1","display_name":"API","description":"tip","sort_order":3,"group_id":null}}`),
		"GET /api/v1/status-pages/p1/components":         jsonResponse(200, `{"components":[{"id":"k1","status_page_id":"p1","check_id":"c1","display_name":"API","description":null,"sort_order":0,"group_id":null}]}`),
		"PATCH /api/v1/status-pages/p1/components/k1":    jsonResponse(200, `{"component":{"id":"k1","status_page_id":"p1","check_id":"c1","display_name":"Public API","description":null,"sort_order":3,"group_id":null}}`),
		"DELETE /api/v1/status-pages/p1/components/k1":   func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) },
	})
	ctx := context.Background()

	created, err := c.CreateStatusPageComponent(ctx, "p1", StatusPageComponentCreate{CheckID: "c1"})
	if err != nil || created.ID != "k1" || created.SortOrder != 3 {
		t.Fatalf("create %+v %v", created, err)
	}
	if b := (*seen)[0].Body; b["checkId"] != "c1" || len(b) != 1 {
		t.Fatalf("create body must be exactly {checkId}: %v", b)
	}

	got, err := c.GetStatusPageComponent(ctx, "p1", "k1")
	if err != nil || got.Description == nil || *got.Description != "tip" {
		t.Fatalf("get %+v %v", got, err)
	}
	list, err := c.ListStatusPageComponents(ctx, "p1")
	if err != nil || len(list) != 1 {
		t.Fatalf("list %v %v", list, err)
	}

	name := "Public API"
	upd, err := c.UpdateStatusPageComponent(ctx, "p1", "k1", StatusPageComponentUpdate{DisplayName: &name, Description: Null()})
	if err != nil || upd.DisplayName != "Public API" {
		t.Fatalf("update %+v %v", upd, err)
	}
	if b := (*seen)[3].Body; b["displayName"] != "Public API" || b["description"] != nil {
		t.Fatalf("update body %v", b)
	}
	if err := c.DeleteStatusPageComponent(ctx, "p1", "k1"); err != nil {
		t.Fatal(err)
	}
}

func TestNotificationChannelCRUDMapping(t *testing.T) {
	c, seen := fakeAPI(t, map[string]func(http.ResponseWriter, *http.Request){
		"POST /api/v1/notification-channels":       jsonResponse(201, `{"channel":{"id":"n1","account_id":"a","type":"discord","name":"Ops","config_summary":"discord.com/api/webhooks/...","verified_at":null,"disabled_at":null,"last_test_at":null,"last_test_ok":null,"last_test_error":null,"created_at":"2026-08-23T00:00:00.000Z","updated_at":"2026-08-23T00:00:00.000Z","provider_configured":true,"type_label":"Discord"}}`),
		"GET /api/v1/notification-channels/n1":     jsonResponse(200, `{"channel":{"id":"n1","type":"discord","name":"Ops","config_summary":"x","disabled_at":"2026-08-23T01:00:00.000Z","provider_configured":true,"type_label":"Discord","created_at":"","updated_at":""}}`),
		"PATCH /api/v1/notification-channels/n1":   jsonResponse(200, `{"channel":{"id":"n1","type":"discord","name":"Ops 2","config_summary":"x","disabled_at":null,"provider_configured":true,"type_label":"Discord","created_at":"","updated_at":""}}`),
		"DELETE /api/v1/notification-channels/n1":  func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) },
		"GET /api/v1/notification-channels":        jsonResponse(200, `{"channels":[]}`),
	})
	ctx := context.Background()

	ch, err := c.CreateNotificationChannel(ctx, NotificationChannelCreate{Type: "discord", Name: "Ops", Config: map[string]string{"webhookUrl": "https://discord.com/api/webhooks/1/x"}})
	if err != nil || ch.ID != "n1" || ch.VerifiedAt != nil {
		t.Fatalf("create %+v %v", ch, err)
	}
	b := (*seen)[0].Body
	if b["type"] != "discord" || b["name"] != "Ops" {
		t.Fatalf("create body %v", b)
	}
	if cfg, ok := b["config"].(map[string]any); !ok || cfg["webhookUrl"] != "https://discord.com/api/webhooks/1/x" {
		t.Fatalf("config body %v", b["config"])
	}

	got, err := c.GetNotificationChannel(ctx, "n1")
	if err != nil || got.DisabledAt == nil {
		t.Fatalf("get %+v %v", got, err)
	}

	enabled := true
	name := "Ops 2"
	upd, err := c.UpdateNotificationChannel(ctx, "n1", NotificationChannelUpdate{Name: &name, Enabled: &enabled})
	if err != nil || upd.Name != "Ops 2" || upd.DisabledAt != nil {
		t.Fatalf("update %+v %v", upd, err)
	}
	if b := (*seen)[2].Body; b["name"] != "Ops 2" || b["enabled"] != true {
		t.Fatalf("update body %v", b)
	}
	if _, present := (*seen)[2].Body["config"]; present {
		t.Fatalf("config must be omitted when unchanged: %v", (*seen)[2].Body)
	}

	if err := c.DeleteNotificationChannel(ctx, "n1"); err != nil {
		t.Fatal(err)
	}
	if list, err := c.ListNotificationChannels(ctx); err != nil || len(list) != 0 {
		t.Fatalf("list %v %v", list, err)
	}
}

func TestCheckAssertionsAndRegionsMapping(t *testing.T) {
	c, seen := fakeAPI(t, map[string]func(http.ResponseWriter, *http.Request){
		"PATCH /api/v1/checks/c1":            jsonResponse(200, `{"check":{"id":"c1","name":"API","type":"http","interval_seconds":60,"selected_regions":["fra","nrt"]}}`),
		"PATCH /api/v1/checks/c1/assertions": jsonResponse(200, `{"check":{"id":"c1","name":"API","type":"http","interval_seconds":60,"selected_regions":["fra"],"assertion_status_min":200,"assertion_status_max":299,"assertion_body_op":"contains","assertion_body_value":"ok","assertion_body_case_sensitive":true}}`),
	})
	ctx := context.Background()

	upd, err := c.UpdateCheckRegions(ctx, "c1", []string{"fra", "nrt"})
	if err != nil || len(upd.SelectedRegions) != 2 {
		t.Fatalf("regions %+v %v", upd, err)
	}
	if b := (*seen)[0].Body; len(b) != 1 || len(b["regions"].([]any)) != 2 {
		t.Fatalf("regions body %v", b)
	}

	op, val := "contains", "ok"
	mn, mx := int64(200), int64(299)
	upd, err = c.UpdateCheckAssertions(ctx, "c1", HTTPAssertions{AssertionBodyOp: &op, AssertionBodyValue: &val, AssertionStatusMin: &mn, AssertionStatusMax: &mx})
	if err != nil || upd.AssertionStatusMin == nil || *upd.AssertionStatusMin != 200 || upd.AssertionBodyCaseSensitive == nil || !*upd.AssertionBodyCaseSensitive {
		t.Fatalf("assertions %+v %v", upd, err)
	}
	b := (*seen)[1].Body
	if b["assertionBodyOp"] != "contains" || b["assertionStatusMin"].(float64) != 200 {
		t.Fatalf("assertions body %v", b)
	}
	if _, present := b["assertionHeaderName"]; present {
		t.Fatalf("unset group must be omitted (full replace clears it): %v", b)
	}
}
