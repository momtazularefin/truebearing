package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// NewRouter builds the top-level HTTP router with middleware and all routes.
func NewRouter(db *pgxpool.Pool, rdb *redis.Client, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	// Global middleware stack.
	r.Use(chimiddleware.RealIP)
	r.Use(RequestID)
	r.Use(RequestLogger(logger))
	r.Use(chimiddleware.Recoverer)

	// Operational probes.
	r.Get("/healthz", HealthHandler(db, rdb))
	r.Get("/readyz", ReadyHandler(db, rdb))

	// API v1 resource routes (stubs).
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/tenants", func(r chi.Router) {
			r.Get("/", notImplemented)
			r.Post("/", notImplemented)
		})

		r.Route("/datasets", func(r chi.Router) {
			r.Get("/", notImplemented)
			r.Post("/", notImplemented)
			r.Get("/{id}", notImplemented)
			r.Put("/{id}", notImplemented)
			r.Delete("/{id}", notImplemented)
		})

		r.Route("/prompts", func(r chi.Router) {
			r.Get("/", notImplemented)
			r.Post("/", notImplemented)
			r.Get("/{id}", notImplemented)
			r.Put("/{id}", notImplemented)
			r.Delete("/{id}", notImplemented)
		})

		r.Route("/runs", func(r chi.Router) {
			r.Get("/", notImplemented)
			r.Post("/", notImplemented)
			r.Get("/{id}", notImplemented)
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
