package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/momtazularefin/truebearing/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestMetricsMiddlewareRecordsRoutePattern(t *testing.T) {
	r := chi.NewRouter()
	r.Use(metrics.Middleware)

	r.Get("/api/v1/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/items/item-12345", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	metricVal := getCounterValue(t, metrics.HTTPRequestsTotal, http.MethodGet, "/api/v1/items/{id}", "200")
	if metricVal < 1 {
		t.Errorf("expected at least 1 request recorded for route pattern /api/v1/items/{id}, got %f", metricVal)
	}
}

func getCounterValue(t *testing.T, counter *prometheus.CounterVec, method, route, status string) float64 {
	c, err := counter.GetMetricWithLabelValues(method, route, status)
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	return m.GetCounter().GetValue()
}
