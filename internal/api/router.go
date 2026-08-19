package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/momtazularefin/truebearing/internal/metrics"
	"github.com/redis/go-redis/v9"
)

// NewRouter builds the top-level HTTP router with middleware and all routes.
func NewRouter(db *pgxpool.Pool, rdb *redis.Client, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	// Global middleware stack (outermost first)
	r.Use(chimiddleware.RealIP)
	r.Use(RequestID)
	r.Use(RequestLogger(logger))
	r.Use(metrics.Middleware)
	r.Use(chimiddleware.Recoverer)

	// Operational probes (unauthenticated)
	r.Get("/healthz", HealthHandler(db, rdb))
	r.Get("/readyz", ReadyHandler(db, rdb))

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public tenant registration (generates initial tenant & API key)
		r.Post("/tenants", CreateTenantHandler(db))

		// Authenticated routes group
		r.Group(func(r chi.Router) {
			r.Use(Authenticate(db))

			// Tenant and API key management
			r.Route("/tenants", func(r chi.Router) {
				r.Get("/me", GetTenantMeHandler(db))
				r.Post("/keys", CreateAPIKeyHandler(db))
				r.Delete("/keys/{id}", RevokeAPIKeyHandler(db))
			})

			// Datasets CRUD
			r.Route("/datasets", func(r chi.Router) {
				r.Get("/", ListDatasetsHandler(db))
				r.Post("/", CreateDatasetHandler(db))
				r.Get("/{id}", GetDatasetHandler(db))
				r.Put("/{id}", UpdateDatasetHandler(db))
				r.Delete("/{id}", DeleteDatasetHandler(db))
			})

			// Prompts CRUD (versioned)
			r.Route("/prompts", func(r chi.Router) {
				r.Get("/", ListPromptsHandler(db))
				r.Post("/", CreatePromptHandler(db))
				r.Get("/{id}", GetPromptHandler(db))
				r.Put("/{id}", UpdatePromptHandler(db))
				r.Delete("/{id}", DeletePromptHandler(db))
			})

			// Runs (M2 queue & workers)
			r.Route("/runs", func(r chi.Router) {
				r.Get("/", notImplemented)
				r.Post("/", notImplemented)
				r.Get("/{id}", notImplemented)
			})
		})
	})

	return r
}

// notImplemented is a placeholder handler that returns 501 Not Implemented.
func notImplemented(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "not implemented",
	})
}
