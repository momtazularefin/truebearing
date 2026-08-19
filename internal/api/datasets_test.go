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

func TestDatasetValidationLogic(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handler := api.NewRouter(nil, nil, logger)

	t.Run("create dataset requires authentication", func(t *testing.T) {
		reqBody := api.CreateDatasetRequest{
			Name:        "eval-set-1",
			Description: "Benchmark dataset",
			Items: []api.DatasetItem{
				{
					ID:    "case-1",
					Input: map[string]any{"query": "hello"},
				},
			},
		}
		raw, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/datasets", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("get dataset by id requires authentication", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/11111111-1111-1111-1111-111111111111", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("put dataset requires authentication", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/v1/datasets/11111111-1111-1111-1111-111111111111", bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("delete dataset requires authentication", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/datasets/11111111-1111-1111-1111-111111111111", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})
}
