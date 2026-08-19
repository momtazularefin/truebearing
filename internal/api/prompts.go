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
	"github.com/momtazularefin/truebearing/internal/database"
)

// Prompt represents a versioned prompt template.
type Prompt struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Name      string    `json:"name"`
	Template  string    `json:"template"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreatePromptRequest is the payload for creating a prompt template.
type CreatePromptRequest struct {
	Name     string `json:"name"`
	Template string `json:"template"`
}

// UpdatePromptRequest is the payload for creating a new version of an existing prompt template.
type UpdatePromptRequest struct {
	Template string `json:"template"`
}

// CreatePromptHandler creates a prompt template.
func CreatePromptHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := TenantFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var req CreatePromptRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "prompt name is required")
			return
		}

		req.Template = strings.TrimSpace(req.Template)
		if req.Template == "" {
			writeError(w, http.StatusBadRequest, "prompt template is required")
			return
		}

		var prompt Prompt
		err := database.WithTenantTx(r.Context(), pool, tenantID, func(tx pgx.Tx) error {
			// Find max version for this prompt name
			var maxVersion int
			err := tx.QueryRow(r.Context(), `
				SELECT COALESCE(MAX(version), 0)
				FROM prompts
				WHERE name = $1
			`, req.Name).Scan(&maxVersion)
			if err != nil {
				return err
			}

			nextVersion := maxVersion + 1

			return tx.QueryRow(r.Context(), `
				INSERT INTO prompts (tenant_id, name, template, version)
				VALUES ($1, $2, $3, $4)
				RETURNING id, tenant_id, name, template, version, created_at, updated_at
			`, tenantID, req.Name, req.Template, nextVersion).Scan(
				&prompt.ID,
				&prompt.TenantID,
				&prompt.Name,
				&prompt.Template,
				&prompt.Version,
				&prompt.CreatedAt,
				&prompt.UpdatedAt,
			)
		})

		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				writeError(w, http.StatusConflict, "prompt version already exists")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to create prompt template")
			return
		}

		writeJSON(w, http.StatusCreated, prompt)
	}
}

// ListPromptsHandler lists prompts for the authenticated tenant.
func ListPromptsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := TenantFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		nameFilter := strings.TrimSpace(r.URL.Query().Get("name"))

		prompts := make([]Prompt, 0)
		err := database.WithTenantTx(r.Context(), pool, tenantID, func(tx pgx.Tx) error {
			var rows pgx.Rows
			var err error

			if nameFilter != "" {
				rows, err = tx.Query(r.Context(), `
					SELECT id, tenant_id, name, template, version, created_at, updated_at
					FROM prompts
					WHERE name = $1
					ORDER BY version DESC
				`, nameFilter)
			} else {
				rows, err = tx.Query(r.Context(), `
					SELECT id, tenant_id, name, template, version, created_at, updated_at
					FROM prompts
					ORDER BY name ASC, version DESC
				`)
			}

			if err != nil {
				return err
			}
			defer rows.Close()

			for rows.Next() {
				var p Prompt
				if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Template, &p.Version, &p.CreatedAt, &p.UpdatedAt); err != nil {
					return err
				}
				prompts = append(prompts, p)
			}
			return rows.Err()
		})

		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list prompts")
			return
		}

		writeJSON(w, http.StatusOK, prompts)
	}
}

// GetPromptHandler retrieves a specific prompt template by ID.
func GetPromptHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := TenantFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid prompt id")
			return
		}

		var prompt Prompt
		err = database.WithTenantTx(r.Context(), pool, tenantID, func(tx pgx.Tx) error {
			return tx.QueryRow(r.Context(), `
				SELECT id, tenant_id, name, template, version, created_at, updated_at
				FROM prompts
				WHERE id = $1
			`, id).Scan(
				&prompt.ID,
				&prompt.TenantID,
				&prompt.Name,
				&prompt.Template,
				&prompt.Version,
				&prompt.CreatedAt,
				&prompt.UpdatedAt,
			)
		})

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, http.StatusNotFound, "prompt not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to retrieve prompt")
			return
		}

		writeJSON(w, http.StatusOK, prompt)
	}
}

// UpdatePromptHandler creates a new version of an existing prompt template.
func UpdatePromptHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := TenantFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid prompt id")
			return
		}

		var req UpdatePromptRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		req.Template = strings.TrimSpace(req.Template)
		if req.Template == "" {
			writeError(w, http.StatusBadRequest, "template is required")
			return
		}

		var newPrompt Prompt
		err = database.WithTenantTx(r.Context(), pool, tenantID, func(tx pgx.Tx) error {
			// Find existing prompt name
			var promptName string
			err := tx.QueryRow(r.Context(), `
				SELECT name
				FROM prompts
				WHERE id = $1
			`, id).Scan(&promptName)
			if err != nil {
				return err
			}

			// Get latest version for this prompt name
			var maxVersion int
			err = tx.QueryRow(r.Context(), `
				SELECT MAX(version)
				FROM prompts
				WHERE name = $1
			`, promptName).Scan(&maxVersion)
			if err != nil {
				return err
			}

			nextVersion := maxVersion + 1

			return tx.QueryRow(r.Context(), `
				INSERT INTO prompts (tenant_id, name, template, version)
				VALUES ($1, $2, $3, $4)
				RETURNING id, tenant_id, name, template, version, created_at, updated_at
			`, tenantID, promptName, req.Template, nextVersion).Scan(
				&newPrompt.ID,
				&newPrompt.TenantID,
				&newPrompt.Name,
				&newPrompt.Template,
				&newPrompt.Version,
				&newPrompt.CreatedAt,
				&newPrompt.UpdatedAt,
			)
		})

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, http.StatusNotFound, "prompt not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to update prompt template")
			return
		}

		writeJSON(w, http.StatusCreated, newPrompt)
	}
}

// DeletePromptHandler deletes a prompt version by ID.
func DeletePromptHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := TenantFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid prompt id")
			return
		}

		var rowsAffected int64
		err = database.WithTenantTx(r.Context(), pool, tenantID, func(tx pgx.Tx) error {
			tag, err := tx.Exec(r.Context(), `
				DELETE FROM prompts
				WHERE id = $1
			`, id)
			if err != nil {
				return err
			}
			rowsAffected = tag.RowsAffected()
			return nil
		})

		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to delete prompt")
			return
		}

		if rowsAffected == 0 {
			writeError(w, http.StatusNotFound, "prompt not found")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}
