// Package metrics provides Prometheus instrumentation and a dedicated metrics HTTP listener.
package metrics

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTPRequestDuration tracks HTTP request latency partitioned by route pattern, method, and status code.
	// Route pattern (e.g. /api/v1/datasets/{id}) is used instead of raw URL path to prevent cardinality explosion.
	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds partitioned by method, route pattern, and status code.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "status"},
	)

	// HTTPRequestsTotal counts total HTTP requests partitioned by method, route pattern, and status code.
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests partitioned by method, route pattern, and status code.",
		},
		[]string{"method", "route", "status"},
	)
)

func init() {
	prometheus.MustRegister(HTTPRequestDuration)
	prometheus.MustRegister(HTTPRequestsTotal)
}

// Middleware records Prometheus metrics for each HTTP request using chi route pattern.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()
		routePattern := chi.RouteContext(r.Context()).RoutePattern()
		if routePattern == "" {
			routePattern = "unmatched"
		}

		statusStr := strconv.Itoa(rw.statusCode)
		method := r.Method

		HTTPRequestDuration.WithLabelValues(method, routePattern, statusStr).Observe(duration)
		HTTPRequestsTotal.WithLabelValues(method, routePattern, statusStr).Inc()
	})
}

// statusResponseWriter captures the status code written to the ResponseWriter.
type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// NewServer creates a dedicated HTTP server for serving /metrics on an internal pod listener.
// Keeping /metrics on a separate listener ensures operational metrics are not exposed through the public ingress.
func NewServer(addr string, logger *slog.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	return &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
}
