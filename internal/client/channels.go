package client

import (
	"context"
	"net/http"
)

// NotificationChannel is the `channel` entity (OpenAPI:
// notificationChannelEntitySchema). It NEVER carries the config: the webhook
// URL or API key is write-only and `ConfigSummary` is the redacted
// description ("discord.com/api/webhooks/...", "EU region, API key stored").
type NotificationChannel struct {
	ID                 string  `json:"id"`
	AccountID          string  `json:"account_id"`
	Type               string  `json:"type"`
	Name               string  `json:"name"`
	ConfigSummary      string  `json:"config_summary"`
	VerifiedAt         *string `json:"verified_at"`
	DisabledAt         *string `json:"disabled_at"`
	LastTestAt         *string `json:"last_test_at"`
	LastTestOK         *bool   `json:"last_test_ok"`
	LastTestError      *string `json:"last_test_error"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
	ProviderConfigured bool    `json:"provider_configured"`
	TypeLabel          string  `json:"type_label"`
}

// NotificationChannelCreate is `notificationChannelCreateSchema`:
// POST /notification-channels. Config keys by type: discord and msteams
// take `webhookUrl`; generic_json takes `url`; opsgenie takes `apiKey` and
// `region` (us|eu).
type NotificationChannelCreate struct {
	Type   string            `json:"type"`
	Name   string            `json:"name"`
	Config map[string]string `json:"config"`
}

// NotificationChannelUpdate is `notificationChannelUpdateSchema`:
// PATCH /notification-channels/{id}. Replacing config clears verification.
type NotificationChannelUpdate struct {
	Name    *string           `json:"name,omitempty"`
	Config  map[string]string `json:"config,omitempty"`
	Enabled *bool             `json:"enabled,omitempty"`
}

type channelEnvelope struct {
	Channel NotificationChannel `json:"channel"`
}

type channelsEnvelope struct {
	Channels []NotificationChannel `json:"channels"`
}

// CreateNotificationChannel: POST /notification-channels (201).
func (c *Client) CreateNotificationChannel(ctx context.Context, in NotificationChannelCreate) (*NotificationChannel, error) {
	var out channelEnvelope
	if err := c.Do(ctx, http.MethodPost, "/notification-channels", in, &out); err != nil {
		return nil, err
	}
	return &out.Channel, nil
}

// GetNotificationChannel: GET /notification-channels/{id}.
func (c *Client) GetNotificationChannel(ctx context.Context, id string) (*NotificationChannel, error) {
	var out channelEnvelope
	if err := c.Do(ctx, http.MethodGet, "/notification-channels/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out.Channel, nil
}

// ListNotificationChannels: GET /notification-channels.
func (c *Client) ListNotificationChannels(ctx context.Context) ([]NotificationChannel, error) {
	var out channelsEnvelope
	if err := c.Do(ctx, http.MethodGet, "/notification-channels", nil, &out); err != nil {
		return nil, err
	}
	return out.Channels, nil
}

// UpdateNotificationChannel: PATCH /notification-channels/{id}.
func (c *Client) UpdateNotificationChannel(ctx context.Context, id string, in NotificationChannelUpdate) (*NotificationChannel, error) {
	var out channelEnvelope
	if err := c.Do(ctx, http.MethodPatch, "/notification-channels/"+id, in, &out); err != nil {
		return nil, err
	}
	return &out.Channel, nil
}

// DeleteNotificationChannel: DELETE /notification-channels/{id} (204).
func (c *Client) DeleteNotificationChannel(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "/notification-channels/"+id, nil, nil)
}
