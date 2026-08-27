// Package client is a tiny hand-written HTTP client for the RealUptime REST
// API v1, covering exactly the routes the Terraform provider needs. It is
// written against the OpenAPI 3.1 document served at
// https://realuptime.io/api/v1/openapi.json (generated from the same zod
// schemas the routes validate with: packages/db/api-schemas.ts in the
// RealUptime monorepo). Field names are the API's own, never renamed.
//
// Authentication is the same bearer key the CLI and MCP server use:
// `Authorization: Bearer ru_live_...`, from the REALUPTIME_API_KEY
// environment variable or the provider block. The REST API is available on
// Growth and Scale plans; a read-only key can read but every Create/Update/
// Delete here needs a read_write key (RU-1004 otherwise).
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultBaseURL is the production API root. The CLI's DEFAULT_API_URL is the
// same string; both surfaces accept REALUPTIME_API_URL to override it (for a
// local dev server in acceptance tests, or a self-hosted mirror).
const DefaultBaseURL = "https://realuptime.io/api/v1"

// UserAgent names the provider in server logs.
var UserAgent = "terraform-provider-realuptime"

// Client talks to one RealUptime account.
type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

// New returns a client for baseURL (trailing slashes trimmed) with apiKey.
func New(baseURL, apiKey string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

// APIError is the `{ error, code, docs }` envelope every v1 refusal answers
// with (apps/web/lib/api-auth.ts's apiAuthErrorResponse and each route's own
// literal error responses). `Code` is a stable RU-XXXX registry identifier
// documented at https://realuptime.io/kb/errors/<code>.
type APIError struct {
	Status  int    `json:"-"`
	Message string `json:"error"`
	Code    string `json:"code"`
	Docs    string `json:"docs"`
	// RetryAfter is the Retry-After header in seconds on a 429, else 0.
	RetryAfter int `json:"-"`
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "RealUptime API returned HTTP %d", e.Status)
	if e.Code != "" {
		fmt.Fprintf(&b, " (%s)", e.Code)
	}
	if e.Message != "" {
		fmt.Fprintf(&b, ": %s", e.Message)
	}
	if e.Docs != "" {
		fmt.Fprintf(&b, " (see %s)", e.Docs)
	}
	return b.String()
}

// IsNotFound reports whether err is a 404 from the API: the resource is gone
// (or was never this account's), which Terraform treats as "remove from
// state" on Read.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound
}

// Do performs one request. `in` (if non-nil) is JSON-encoded as the body;
// `out` (if non-nil) receives the decoded 2xx response body. A 204 is a
// success with no body. Any non-2xx is returned as *APIError.
//
// A 429 is retried once after the server's Retry-After, because the write
// budget is per key per minute and an `apply` that creates many resources
// at once can trip it legitimately; anything beyond one retry is surfaced
// so the operator can lower -parallelism rather than have the provider sit
// in a loop.
func (c *Client) Do(ctx context.Context, method, path string, in, out any) error {
	err := c.doOnce(ctx, method, path, in, out)
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.Status == http.StatusTooManyRequests && apiErr.RetryAfter > 0 && apiErr.RetryAfter <= 60 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(apiErr.RetryAfter) * time.Second):
		}
		return c.doOnce(ctx, method, path, in, out)
	}
	return err
}

func (c *Client) doOnce(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		buf, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
		body = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	res, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("%s %s: reading response: %w", method, path, err)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		apiErr := &APIError{Status: res.StatusCode}
		if len(raw) > 0 {
			// Best effort: a proxy's HTML error page is not JSON, and the
			// status code alone is still worth reporting.
			_ = json.Unmarshal(raw, apiErr)
		}
		if apiErr.Message == "" {
			apiErr.Message = strings.TrimSpace(string(raw))
			if len(apiErr.Message) > 300 {
				apiErr.Message = apiErr.Message[:300] + "..."
			}
		}
		if ra := res.Header.Get("Retry-After"); ra != "" {
			fmt.Sscanf(ra, "%d", &apiErr.RetryAfter)
		}
		return apiErr
	}

	if out == nil || res.StatusCode == http.StatusNoContent || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%s %s: decoding response: %w", method, path, err)
	}
	return nil
}
