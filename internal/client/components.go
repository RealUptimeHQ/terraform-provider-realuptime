package client

import (
	"context"
	"net/http"
)

// StatusPageComponent is the `component` entity (OpenAPI:
// statusPageComponentEntitySchema): a check published on a status page.
type StatusPageComponent struct {
	ID           string  `json:"id"`
	StatusPageID string  `json:"status_page_id"`
	CheckID      string  `json:"check_id"`
	DisplayName  string  `json:"display_name"`
	Description  *string `json:"description"`
	SortOrder    int64   `json:"sort_order"`
	GroupID      *string `json:"group_id"`
}

// StatusPageComponentCreate is `statusPageComponentCreateSchema`:
// POST /status-pages/{id}/components. displayName defaults to the check's
// own name when omitted.
type StatusPageComponentCreate struct {
	CheckID     string         `json:"checkId"`
	DisplayName *string        `json:"displayName,omitempty"`
	Description NullableString `json:"description,omitzero"`
}

// StatusPageComponentUpdate is `statusPageComponentUpdateSchema`:
// PATCH /status-pages/{id}/components/{componentId}.
type StatusPageComponentUpdate struct {
	DisplayName *string        `json:"displayName,omitempty"`
	Description NullableString `json:"description,omitzero"`
}

type componentEnvelope struct {
	Component StatusPageComponent `json:"component"`
}

type componentsEnvelope struct {
	Components []StatusPageComponent `json:"components"`
}

// CreateStatusPageComponent: POST /status-pages/{id}/components (201).
func (c *Client) CreateStatusPageComponent(ctx context.Context, statusPageID string, in StatusPageComponentCreate) (*StatusPageComponent, error) {
	var out componentEnvelope
	if err := c.Do(ctx, http.MethodPost, "/status-pages/"+statusPageID+"/components", in, &out); err != nil {
		return nil, err
	}
	return &out.Component, nil
}

// GetStatusPageComponent: GET /status-pages/{id}/components/{componentId}.
func (c *Client) GetStatusPageComponent(ctx context.Context, statusPageID, componentID string) (*StatusPageComponent, error) {
	var out componentEnvelope
	if err := c.Do(ctx, http.MethodGet, "/status-pages/"+statusPageID+"/components/"+componentID, nil, &out); err != nil {
		return nil, err
	}
	return &out.Component, nil
}

// ListStatusPageComponents: GET /status-pages/{id}/components, display order.
func (c *Client) ListStatusPageComponents(ctx context.Context, statusPageID string) ([]StatusPageComponent, error) {
	var out componentsEnvelope
	if err := c.Do(ctx, http.MethodGet, "/status-pages/"+statusPageID+"/components", nil, &out); err != nil {
		return nil, err
	}
	return out.Components, nil
}

// UpdateStatusPageComponent: PATCH /status-pages/{id}/components/{componentId}.
func (c *Client) UpdateStatusPageComponent(ctx context.Context, statusPageID, componentID string, in StatusPageComponentUpdate) (*StatusPageComponent, error) {
	var out componentEnvelope
	if err := c.Do(ctx, http.MethodPatch, "/status-pages/"+statusPageID+"/components/"+componentID, in, &out); err != nil {
		return nil, err
	}
	return &out.Component, nil
}

// DeleteStatusPageComponent: DELETE /status-pages/{id}/components/{componentId} (204).
func (c *Client) DeleteStatusPageComponent(ctx context.Context, statusPageID, componentID string) error {
	return c.Do(ctx, http.MethodDelete, "/status-pages/"+statusPageID+"/components/"+componentID, nil, nil)
}
