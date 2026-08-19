package api_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/momtazularefin/truebearing/internal/api"
)

func TestUnauthenticatedRequestsReturn401(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	// NewRouter with nil db/rdb for routing tests
	handler := api.NewRouter(nil, nil, logger)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/tenants/me"},
		{http.MethodPost, "/api/v1/tenants/keys"},
		{http.MethodDelete, "/api/v1/tenants/keys/00000000-0000-0000-0000-000000000000"},
		{http.MethodGet, "/api/v1/datasets"},
		{http.MethodPost, "/api/v1/datasets"},
		{http.MethodGet, "/api/v1/datasets/00000000-0000-0000-0000-000000000000"},
		{http.MethodPut, "/api/v1/datasets/00000000-0000-0000-0000-000000000000"},
		{http.MethodDelete, "/api/v1/datasets/00000000-0000-0000-0000-000000000000"},
		{http.MethodGet, "/api/v1/prompts"},
		{http.MethodPost, "/api/v1/prompts"},
		{http.MethodGet, "/api/v1/prompts/00000000-0000-0000-0000-000000000000"},
		{http.MethodPut, "/api/v1/prompts/00000000-0000-0000-0000-000000000000"},
		{http.MethodDelete, "/api/v1/prompts/00000000-0000-0000-0000-000000000000"},
		{http.MethodGet, "/api/v1/runs"},
		{http.MethodPost, "/api/v1/runs"},
		{http.MethodGet, "/api/v1/runs/00000000-0000-0000-0000-000000000000"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status %d for unauthenticated request, got %d", http.StatusUnauthorized, rec.Code)
			}

			var body map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode response body: %v", err)
			}
			if body["error"] != "unauthorized" {
				t.Errorf("expected error message %q, got %q", "unauthorized", body["error"])
			}
		})
	}
}

func TestMalformedAuthHeadersReturn401(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handler := api.NewRouter(nil, nil, logger)

	malformedHeaders := []string{
		"",
		"Basic dXNlcjpwYXNz",
		"Bearer ",
		"Bearer invalid_key_format",
		"Bearer tb_short",
	}

	for _, h := range malformedHeaders {
		t.Run("header_"+h, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/datasets", nil)
			if h != "" {
				req.Header.Set("Authorization", h)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status %d for header %q, got %d", http.StatusUnauthorized, h, rec.Code)
			}
		})
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handler := api.NewRouter(nil, nil, logger)

	t.Run("generates new request ID if missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		respID := rec.Header().Get("X-Request-ID")
		if respID == "" {
			t.Error("expected X-Request-ID header in response")
		}
	})

	t.Run("preserves incoming request ID", func(t *testing.T) {
		const customID = "custom-req-id-12345"
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.Header.Set("X-Request-ID", customID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		respID := rec.Header().Get("X-Request-ID")
		if respID != customID {
			t.Errorf("expected X-Request-ID %q, got %q", customID, respID)
		}
	})
}
