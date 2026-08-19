package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/momtazularefin/truebearing/internal/api"
)

func TestPromptValidationLogic(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handler := api.NewRouter(nil, nil, logger)

	t.Run("create prompt requires authentication", func(t *testing.T) {
		reqBody := api.CreatePromptRequest{
			Name:     "judge-rubric",
			Template: "Evaluate response: {{response}} based on: {{rubric}}",
		}
		raw, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/prompts", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("get prompt by id requires authentication", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/prompts/22222222-2222-2222-2222-222222222222", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("put prompt requires authentication", func(t *testing.T) {
		reqBody := api.UpdatePromptRequest{
			Template: "Updated template: {{response}}",
		}
		raw, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/prompts/22222222-2222-2222-2222-222222222222", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("delete prompt requires authentication", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/prompts/22222222-2222-2222-2222-222222222222", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})
}
