package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/momtazularefin/truebearing/internal/auth"
)

// contextKey is an unexported type for context keys in this package.
type contextKey string

const (
	requestIDKey contextKey = "request_id"
	tenantIDKey  contextKey = "tenant_id"
	apiKeyIDKey  contextKey = "api_key_id"
)

// RequestID is HTTP middleware that ensures every request carries a unique ID.
// It first checks the X-Request-ID header; if absent, it generates a new UUID.
// The ID is set on the response header and stored in the request context.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestLogger returns middleware that logs each request's method, path,
// status code, duration, and request ID using the provided structured logger.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)
			duration := time.Since(start)

			reqID, _ := r.Context().Value(requestIDKey).(string)

			logger.Info("request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.statusCode,
				"duration_ms", duration.Milliseconds(),
				"request_id", reqID,
			)
		})
	}
}

// Authenticate returns middleware that validates the Bearer API key against the database.
//
// Per AC1 and D003/D009, authentication returns a uniform 401 Unauthorized with no timing
// or response distinction between an unknown key, an inactive key, or an expired key.
func Authenticate(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			rawKey := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			if err := auth.ValidateKeyFormat(rawKey); err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			keyHash := auth.HashAPIKey(rawKey)

			var keyID, tenantID uuid.UUID
			query := `
				SELECT id, tenant_id
				FROM api_keys
				WHERE key_hash = $1
				  AND is_active = TRUE
				  AND (expires_at IS NULL OR expires_at > NOW())
			`
			err := pool.QueryRow(r.Context(), query, keyHash).Scan(&keyID, &tenantID)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			ctx := context.WithValue(r.Context(), tenantIDKey, tenantID)
			ctx = context.WithValue(ctx, apiKeyIDKey, keyID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TenantFromContext extracts the authenticated tenant UUID from the request context.
func TenantFromContext(ctx context.Context) (uuid.UUID, bool) {
	val, ok := ctx.Value(tenantIDKey).(uuid.UUID)
	return val, ok
}

// APIKeyFromContext extracts the authenticated API key UUID from the request context.
func APIKeyFromContext(ctx context.Context) (uuid.UUID, bool) {
	val, ok := ctx.Value(apiKeyIDKey).(uuid.UUID)
	return val, ok
}

// writeJSON writes a JSON payload with the specified HTTP status code.
func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError writes a standardized JSON error message.
func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and delegates to the underlying writer.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
