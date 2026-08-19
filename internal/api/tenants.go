package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/momtazularefin/truebearing/internal/auth"
	"github.com/momtazularefin/truebearing/internal/database"
)

// TenantResponse represents the public tenant entity returned by the API.
type TenantResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	APIKey    string    `json:"api_key,omitempty"` // Only populated on creation
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateTenantRequest is the payload for registering a new tenant.
type CreateTenantRequest struct {
	Name string `json:"name"`
}

// CreateKeyRequest is the payload for generating an additional API key.
type CreateKeyRequest struct {
	Name string `json:"name"`
}

// KeyResponse represents an API key response.
type KeyResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	APIKey    string    `json:"api_key,omitempty"` // Plaintext key returned only upon creation
	CreatedAt time.Time `json:"created_at"`
}

// CreateTenantHandler registers a new tenant and generates its initial API key.
// Per D009, the raw API key is returned in the response exactly once.
func CreateTenantHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateTenantRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "tenant name is required")
			return
		}

		plainKey, keyHash, err := auth.GenerateAPIKey()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to generate api key")
			return
		}

		ctx := r.Context()
		tx, err := pool.Begin(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to begin transaction")
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		var tenant TenantResponse
		err = tx.QueryRow(ctx, `
			INSERT INTO tenants (name)
			VALUES ($1)
			RETURNING id, name, created_at, updated_at
		`, req.Name).Scan(&tenant.ID, &tenant.Name, &tenant.CreatedAt, &tenant.UpdatedAt)

		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique violation
				writeError(w, http.StatusConflict, "tenant with this name already exists")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to create tenant")
			return
		}

		// Insert the initial API key for the tenant
		_, err = tx.Exec(ctx, `
			INSERT INTO api_keys (tenant_id, key_hash, name, is_active)
			VALUES ($1, $2, $3, TRUE)
		`, tenant.ID, keyHash, "default")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create initial api key")
			return
		}

		if err := tx.Commit(ctx); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to commit tenant creation")
			return
		}

		tenant.APIKey = plainKey
		writeJSON(w, http.StatusCreated, tenant)
	}
}

// GetTenantMeHandler retrieves the authenticated tenant's information.
func GetTenantMeHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := TenantFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var tenant TenantResponse
		err := pool.QueryRow(r.Context(), `
			SELECT id, name, created_at, updated_at
			FROM tenants
			WHERE id = $1
		`, tenantID).Scan(&tenant.ID, &tenant.Name, &tenant.CreatedAt, &tenant.UpdatedAt)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, http.StatusNotFound, "tenant not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to query tenant")
			return
		}

		writeJSON(w, http.StatusOK, tenant)
	}
}

// CreateAPIKeyHandler generates a new API key for the authenticated tenant.
func CreateAPIKeyHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := TenantFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var req CreateKeyRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			req.Name = "api-key"
		}

		plainKey, keyHash, err := auth.GenerateAPIKey()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to generate key")
			return
		}

		var resp KeyResponse
		resp.APIKey = plainKey
		resp.Name = req.Name

		err = database.WithTenantTx(r.Context(), pool, tenantID, func(tx pgx.Tx) error {
			return tx.QueryRow(r.Context(), `
				INSERT INTO api_keys (tenant_id, key_hash, name, is_active)
				VALUES ($1, $2, $3, TRUE)
				RETURNING id, created_at
			`, tenantID, keyHash, req.Name).Scan(&resp.ID, &resp.CreatedAt)
		})

		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create api key")
			return
		}

		writeJSON(w, http.StatusCreated, resp)
	}
}

// RevokeAPIKeyHandler deactivates an API key belonging to the authenticated tenant.
func RevokeAPIKeyHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := TenantFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		keyIDStr := chi.URLParam(r, "id")
		keyID, err := uuid.Parse(keyIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid key id")
			return
		}

		var rowsAffected int64
		err = database.WithTenantTx(r.Context(), pool, tenantID, func(tx pgx.Tx) error {
			tag, err := tx.Exec(r.Context(), `
				UPDATE api_keys
				SET is_active = FALSE
				WHERE id = $1 AND tenant_id = $2
			`, keyID, tenantID)
			if err != nil {
				return err
			}
			rowsAffected = tag.RowsAffected()
			return nil
		})

		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to revoke key")
			return
		}

		if rowsAffected == 0 {
			writeError(w, http.StatusNotFound, "api key not found")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
	}
}
