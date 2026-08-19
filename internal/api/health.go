package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// HealthHandler returns an HTTP handler that reports the health of downstream
// services (Postgres and Redis). If either service is unreachable the overall
// status is "degraded" and a 503 is returned; otherwise status is "ok" / 200.
func HealthHandler(db *pgxpool.Pool, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		pgStatus := "ok"
		if db == nil {
			pgStatus = "error: database connection uninitialized"
		} else if err := db.Ping(ctx); err != nil {
			pgStatus = "error: " + err.Error()
		}

		redisStatus := "ok"
		if rdb == nil {
			redisStatus = "error: redis connection uninitialized"
		} else if err := rdb.Ping(ctx).Err(); err != nil {
			redisStatus = "error: " + err.Error()
		}

		status := "ok"
		httpCode := http.StatusOK
		if pgStatus != "ok" || redisStatus != "ok" {
			status = "degraded"
			httpCode = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(httpCode)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": status,
			"services": map[string]string{
				"postgres": pgStatus,
				"redis":    redisStatus,
			},
		})
	}
}

// ReadyHandler returns an HTTP handler that checks whether the application is
// ready to serve traffic. Both Postgres and Redis must be reachable. Returns
// 200 with {"status":"ready"} on success, or 503 with {"status":"not ready"}.
func ReadyHandler(db *pgxpool.Pool, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var pgErr, redisErr error
		if db == nil {
			pgErr = errors.New("database connection uninitialized")
		} else {
			pgErr = db.Ping(ctx)
		}

		if rdb == nil {
			redisErr = errors.New("redis connection uninitialized")
		} else {
			redisErr = rdb.Ping(ctx).Err()
		}

		w.Header().Set("Content-Type", "application/json")

		if pgErr != nil || redisErr != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "not ready",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "ready",
		})
	}
}
