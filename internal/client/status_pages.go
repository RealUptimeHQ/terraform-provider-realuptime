package client

import (
	"context"
	"net/http"
)

// StatusPage is the `statusPage` entity (OpenAPI: statusPageEntitySchema,
// docs/api.md "GET /status-pages"). Nullable columns are pointers.
type StatusPage struct {
	ID              string  `json:"id"`
	AccountID       string  `json:"account_id"`
	Slug            string  `json:"slug"`
	Name            string  `json:"name"`
	CustomDomain    *string `json:"custom_domain"`
	PageTitle       *string `json:"page_title"`
	PageDescription *string `json:"page_description"`
	LogoURL         *string `json:"logo_url"`
	DomainStatus    string  `json:"domain_status"`
	DomainError     *string `json:"domain_error"`
	HideBranding    bool    `json:"hide_branding"`
	AccentColor     *string `json:"accent_color"`
	HomepageURL     *string `json:"homepage_url"`
	SupportURL      *string `json:"support_url"`
	ThemePreset     *string `json:"theme_preset"`
	FontChoice      *string `json:"font_choice"`
	Locale          *string `json:"locale"`
}

// StatusPageCreate is `statusPageCreateSchema`: POST /status-pages.
type StatusPageCreate struct {
	Name string  `json:"name"`
	Slug *string `json:"slug,omitempty"`
}

// StatusPageUpdate is `statusPageUpdateSchema`: PATCH /status-pages/{id}.
// Every field optional. The API distinguishes "omitted" (leave alone) from
// "null" (clear), so nullable fields are encoded through NullableString:
// absent from the JSON when unset, `null` when explicitly cleared.
type StatusPageUpdate struct {
	Name            *string        `json:"name,omitempty"`
	Slug            *string        `json:"slug,omitempty"`
	PageTitle       NullableString `json:"pageTitle,omitzero"`
	PageDescription NullableString `json:"pageDescription,omitzero"`
	LogoURL         NullableString `json:"logoUrl,omitzero"`
	HideBranding    *bool          `json:"hideBranding,omitempty"`
	AccentColor     NullableString `json:"accentColor,omitzero"`
	HomepageURL     NullableString `json:"homepageUrl,omitzero"`
	SupportURL      NullableString `json:"supportUrl,omitzero"`
	ThemePreset     NullableString `json:"themePreset,omitzero"`
	FontChoice      NullableString `json:"fontChoice,omitzero"`
	Locale          NullableString `json:"locale,omitzero"`
}

type statusPageEnvelope struct {
	StatusPage StatusPage `json:"statusPage"`
}

type statusPagesEnvelope struct {
	StatusPages []StatusPage `json:"statusPages"`
}

// CreateStatusPage: POST /status-pages (201).
func (c *Client) CreateStatusPage(ctx context.Context, in StatusPageCreate) (*StatusPage, error) {
	var out statusPageEnvelope
	if err := c.Do(ctx, http.MethodPost, "/status-pages", in, &out); err != nil {
		return nil, err
	}
	return &out.StatusPage, nil
}

// GetStatusPage: GET /status-pages/{id}.
func (c *Client) GetStatusPage(ctx context.Context, id string) (*StatusPage, error) {
	var out statusPageEnvelope
	if err := c.Do(ctx, http.MethodGet, "/status-pages/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out.StatusPage, nil
}

// ListStatusPages: GET /status-pages, oldest first.
func (c *Client) ListStatusPages(ctx context.Context) ([]StatusPage, error) {
	var out statusPagesEnvelope
	if err := c.Do(ctx, http.MethodGet, "/status-pages", nil, &out); err != nil {
		return nil, err
	}
	return out.StatusPages, nil
}

// UpdateStatusPage: PATCH /status-pages/{id}.
func (c *Client) UpdateStatusPage(ctx context.Context, id string, in StatusPageUpdate) (*StatusPage, error) {
	var out statusPageEnvelope
	if err := c.Do(ctx, http.MethodPatch, "/status-pages/"+id, in, &out); err != nil {
		return nil, err
	}
	return &out.StatusPage, nil
}

// DeleteStatusPage: DELETE /status-pages/{id} (204). The account's last
// page is refused with RU-5011 (409).
func (c *Client) DeleteStatusPage(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "/status-pages/"+id, nil, nil)
}
