package client

import (
	"context"
	"net/http"
)

// Check is the `check` entity (OpenAPI: checkEntitySchema, docs/api.md
// "GET /checks"). Response fields are snake_case; pointers mark nullable
// columns. Only the fields the provider manages are declared; the API may
// send more (tcp_*, smtp_*, multistep steps) and they are ignored.
type Check struct {
	ID                        string   `json:"id"`
	AccountID                 string   `json:"account_id"`
	Name                      string   `json:"name"`
	URL                       *string  `json:"url"`
	Type                      string   `json:"type"`
	IntervalSeconds           int64    `json:"interval_seconds"`
	SelectedRegions           []string `json:"selected_regions"`
	AssertionBodyOp           *string  `json:"assertion_body_op"`
	AssertionBodyValue        *string  `json:"assertion_body_value"`
	AssertionBodyCaseSensitive *bool   `json:"assertion_body_case_sensitive"`
	AssertionHeaderName       *string  `json:"assertion_header_name"`
	AssertionHeaderOp         *string  `json:"assertion_header_op"`
	AssertionHeaderValue      *string  `json:"assertion_header_value"`
	AssertionStatusMin        *int64   `json:"assertion_status_min"`
	AssertionStatusMax        *int64   `json:"assertion_status_max"`
	AssertionJSONPath         *string  `json:"assertion_json_path"`
	AssertionJSONOp           *string  `json:"assertion_json_op"`
	AssertionJSONValue        *string  `json:"assertion_json_value"`
}

// HTTPAssertions is `httpAssertionsShape` from packages/db/api-schemas.ts:
// the request-side (camelCase) assertion fields, shared by POST /checks and
// PATCH /checks/{id}/assertions. Every field is optional; a group (body,
// header, status, json) is set together or omitted together, and PATCH is a
// FULL REPLACE, so an omitted group clears it.
type HTTPAssertions struct {
	AssertionBodyOp            *string `json:"assertionBodyOp,omitempty"`
	AssertionBodyValue         *string `json:"assertionBodyValue,omitempty"`
	AssertionBodyCaseSensitive *bool   `json:"assertionBodyCaseSensitive,omitempty"`
	AssertionHeaderName        *string `json:"assertionHeaderName,omitempty"`
	AssertionHeaderOp          *string `json:"assertionHeaderOp,omitempty"`
	AssertionHeaderValue       *string `json:"assertionHeaderValue,omitempty"`
	AssertionStatusMin         *int64  `json:"assertionStatusMin,omitempty"`
	AssertionStatusMax         *int64  `json:"assertionStatusMax,omitempty"`
	AssertionJSONPath          *string `json:"assertionJsonPath,omitempty"`
	AssertionJSONOp            *string `json:"assertionJsonOp,omitempty"`
	AssertionJSONValue         *string `json:"assertionJsonValue,omitempty"`
}

// CheckCreate is `checkCreateSchema`: POST /checks creates an http monitor.
type CheckCreate struct {
	Name            string   `json:"name"`
	URL             string   `json:"url"`
	IntervalSeconds *int64   `json:"intervalSeconds,omitempty"`
	Regions         []string `json:"regions,omitempty"`
	HTTPAssertions
}

type checkEnvelope struct {
	Check Check `json:"check"`
}

type checksEnvelope struct {
	Checks []Check `json:"checks"`
}

// CreateCheck: POST /checks (201).
func (c *Client) CreateCheck(ctx context.Context, in CheckCreate) (*Check, error) {
	var out checkEnvelope
	if err := c.Do(ctx, http.MethodPost, "/checks", in, &out); err != nil {
		return nil, err
	}
	return &out.Check, nil
}

// GetCheck: GET /checks/{id}. The response also carries `status` and
// `regions` (live state), which the provider does not manage.
func (c *Client) GetCheck(ctx context.Context, id string) (*Check, error) {
	var out checkEnvelope
	if err := c.Do(ctx, http.MethodGet, "/checks/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out.Check, nil
}

// ListChecks: GET /checks.
func (c *Client) ListChecks(ctx context.Context) ([]Check, error) {
	var out checksEnvelope
	if err := c.Do(ctx, http.MethodGet, "/checks", nil, &out); err != nil {
		return nil, err
	}
	return out.Checks, nil
}

// CheckUpdate is `checkUpdateSchema` (packages/db/api-schemas.ts): a sparse
// patch for PATCH /checks/{id} (REA-226 gap-list add). Every field is
// optional and `omitempty` -- a field left nil is left unchanged server-side,
// so a caller sends only what actually changed rather than always
// resubmitting the full set. `Regions` keeps the same shape
// `UpdateCheckRegions` always sent.
type CheckUpdate struct {
	Name            *string  `json:"name,omitempty"`
	URL             *string  `json:"url,omitempty"`
	IntervalSeconds *int64   `json:"intervalSeconds,omitempty"`
	Regions         []string `json:"regions,omitempty"`
}

// UpdateCheck: PATCH /checks/{id} (docs/api.md "PATCH /checks/:id"). Name,
// url, interval and regions are all updated in place now (REA-226); the
// resource's plan modifiers decide which fields actually get sent for a
// given plan diff.
func (c *Client) UpdateCheck(ctx context.Context, id string, in CheckUpdate) (*Check, error) {
	var out checkEnvelope
	if err := c.Do(ctx, http.MethodPatch, "/checks/"+id, in, &out); err != nil {
		return nil, err
	}
	return &out.Check, nil
}

// UpdateCheckRegions: PATCH /checks/{id} with `{ regions }` only. Kept as a
// thin wrapper over UpdateCheck for callers (and tests) that only ever
// touched regions before REA-226 widened the route.
func (c *Client) UpdateCheckRegions(ctx context.Context, id string, regions []string) (*Check, error) {
	return c.UpdateCheck(ctx, id, CheckUpdate{Regions: regions})
}

// UpdateCheckAssertions: PATCH /checks/{id}/assertions (full replace).
func (c *Client) UpdateCheckAssertions(ctx context.Context, id string, in HTTPAssertions) (*Check, error) {
	var out checkEnvelope
	if err := c.Do(ctx, http.MethodPatch, "/checks/"+id+"/assertions", in, &out); err != nil {
		return nil, err
	}
	return &out.Check, nil
}

// DeleteCheck: DELETE /checks/{id} (204). Removes the monitor and its history.
func (c *Client) DeleteCheck(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "/checks/"+id, nil, nil)
}
