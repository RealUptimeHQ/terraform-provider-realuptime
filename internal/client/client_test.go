package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// recorded is one request the fake API saw.
type recorded struct {
	Method string
	Path   string
	Auth   string
	Body   map[string]any
}

// fakeAPI is an httptest server that records every request and answers
// from a script keyed by "METHOD /path".
func fakeAPI(t *testing.T, script map[string]func(w http.ResponseWriter, r *http.Request)) (*Client, *[]recorded) {
	t.Helper()
	var seen []recorded
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := recorded{Method: r.Method, Path: r.URL.Path, Auth: r.Header.Get("Authorization")}
		raw, _ := io.ReadAll(r.Body)
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &rec.Body)
		}
		seen = append(seen, rec)
		h, ok := script[r.Method+" "+r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"Not found.","code":"RU-5009","docs":"https://realuptime.io/kb/errors/ru-5009"}`))
			return
		}
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	return New(srv.URL+"/api/v1/", "ru_live_test"), &seen
}

func jsonResponse(status int, body string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func TestNewTrimsTrailingSlashAndDefaultsBaseURL(t *testing.T) {
	if got := New("", "k").BaseURL; got != DefaultBaseURL {
		t.Fatalf("default base url = %q", got)
	}
	if got := New("https://x.example/api/v1///", "k").BaseURL; got != "https://x.example/api/v1" {
		t.Fatalf("trimmed base url = %q", got)
	}
}

func TestDoSendsBearerAndJSON(t *testing.T) {
	c, seen := fakeAPI(t, map[string]func(http.ResponseWriter, *http.Request){
		"POST /api/v1/checks": jsonResponse(201, `{"check":{"id":"c1","name":"API","url":"https://e.x","type":"http","interval_seconds":60,"selected_regions":["iad"]}}`),
	})
	interval := int64(60)
	got, err := c.CreateCheck(context.Background(), CheckCreate{Name: "API", URL: "https://e.x", IntervalSeconds: &interval, Regions: []string{"iad"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "c1" || got.IntervalSeconds != 60 || len(got.SelectedRegions) != 1 {
		t.Fatalf("unexpected check %+v", got)
	}
	r := (*seen)[0]
	if r.Auth != "Bearer ru_live_test" {
		t.Fatalf("auth header = %q", r.Auth)
	}
	if r.Body["name"] != "API" || r.Body["url"] != "https://e.x" || r.Body["intervalSeconds"].(float64) != 60 {
		t.Fatalf("body = %v", r.Body)
	}
	if _, present := r.Body["assertionBodyOp"]; present {
		t.Fatalf("omitted assertion must not be sent: %v", r.Body)
	}
}

func TestDoMapsErrorEnvelope(t *testing.T) {
	c, _ := fakeAPI(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /api/v1/checks/nope": jsonResponse(404, `{"error":"Check not found.","code":"RU-3003","docs":"https://realuptime.io/kb/errors/ru-3003"}`),
	})
	_, err := c.GetCheck(context.Background(), "nope")
	if !IsNotFound(err) {
		t.Fatalf("expected not-found, got %v", err)
	}
	msg := err.Error()
	for _, want := range []string{"404", "RU-3003", "Check not found.", "kb/errors/ru-3003"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q missing %q", msg, want)
		}
	}
}

func TestDoRetriesOnceOn429(t *testing.T) {
	calls := 0
	c, _ := fakeAPI(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /api/v1/status-pages": func(w http.ResponseWriter, _ *http.Request) {
			calls++
			if calls == 1 {
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(429)
				_, _ = w.Write([]byte(`{"error":"Rate limit exceeded.","code":"RU-2002"}`))
				return
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"statusPages":[{"id":"p1","slug":"s","name":"n","domain_status":"none"}]}`))
		},
	})
	pages, err := c.ListStatusPages(context.Background())
	if err != nil || len(pages) != 1 || calls != 2 {
		t.Fatalf("pages=%v err=%v calls=%d", pages, err, calls)
	}
}

func TestDoHandlesNoContent(t *testing.T) {
	c, _ := fakeAPI(t, map[string]func(http.ResponseWriter, *http.Request){
		"DELETE /api/v1/checks/c1": func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) },
	})
	if err := c.DeleteCheck(context.Background(), "c1"); err != nil {
		t.Fatal(err)
	}
}

func TestDoNonJSONErrorBody(t *testing.T) {
	c, _ := fakeAPI(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /api/v1/checks": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(502)
			_, _ = w.Write([]byte("<html>bad gateway</html>"))
		},
	})
	_, err := c.ListChecks(context.Background())
	var apiErr *APIError
	if !asAPIError(err, &apiErr) || apiErr.Status != 502 || !strings.Contains(apiErr.Message, "bad gateway") {
		t.Fatalf("err=%v", err)
	}
}

func asAPIError(err error, target **APIError) bool {
	e, ok := err.(*APIError)
	if ok {
		*target = e
	}
	return ok
}
