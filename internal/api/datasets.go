package api

import (
	"encoding/json"
	"errors"
	"fmt"
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

// DatasetItem represents an individual evaluation test case within a dataset.
type DatasetItem struct {
	ID       string         `json:"id"`
	Input    map[string]any `json:"input"`
	Expected any            `json:"expected,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Dataset represents an evaluation dataset record.
type Dataset struct {
	ID          uuid.UUID     `json:"id"`
	TenantID    uuid.UUID     `json:"tenant_id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	ItemCount   int           `json:"item_count,omitempty"`
	Items       []DatasetItem `json:"items,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// CreateDatasetRequest is the payload for creating a new dataset.
type CreateDatasetRequest struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Items       []DatasetItem `json:"items"`
}

// UpdateDatasetRequest is the payload for updating an existing dataset.
type UpdateDatasetRequest struct {
	Name        *string        `json:"name"`
	Description *string        `json:"description"`
	Items       *[]DatasetItem `json:"items"`
}

// validateDatasetItems validates dataset items integrity.
func validateDatasetItems(items []DatasetItem) error {
	seenIDs := make(map[string]bool, len(items))
	for i, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("item at index %d is missing 'id'", i)
		}
		if seenIDs[item.ID] {
			return fmt.Errorf("duplicate item id %q at index %d", item.ID, i)
		}
		seenIDs[item.ID] = true

		if item.Input == nil {
			return fmt.Errorf("item %q is missing 'input' object", item.ID)
		}
	}
	return nil
}

// CreateDatasetHandler creates a new dataset for the authenticated tenant.
func CreateDatasetHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := TenantFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var req CreateDatasetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "dataset name is required")
			return
		}

		if req.Items == nil {
			req.Items = []DatasetItem{}
		} else {
			if err := validateDatasetItems(req.Items); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
		}

		itemsJSON, err := json.Marshal(req.Items)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to encode items")
			return
		}

		var dataset Dataset
		var returnedItemsJSON []byte

		err = database.WithTenantTx(r.Context(), pool, tenantID, func(tx pgx.Tx) error {
			return tx.QueryRow(r.Context(), `
				INSERT INTO datasets (tenant_id, name, description, items)
				VALUES ($1, $2, $3, $4)
				RETURNING id, tenant_id, name, description, items, created_at, updated_at
			`, tenantID, req.Name, req.Description, itemsJSON).Scan(
				&dataset.ID,
				&dataset.TenantID,
				&dataset.Name,
				&dataset.Description,
				&returnedItemsJSON,
				&dataset.CreatedAt,
				&dataset.UpdatedAt,
			)
		})

		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique violation on (tenant_id, name)
				writeError(w, http.StatusConflict, "dataset with this name already exists")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to create dataset")
			return
		}

		_ = json.Unmarshal(returnedItemsJSON, &dataset.Items)
		dataset.ItemCount = len(dataset.Items)

		writeJSON(w, http.StatusCreated, dataset)
	}
}

// ListDatasetsHandler lists all datasets for the authenticated tenant.
func ListDatasetsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := TenantFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		datasets := make([]Dataset, 0)
		err := database.WithTenantTx(r.Context(), pool, tenantID, func(tx pgx.Tx) error {
			rows, err := tx.Query(r.Context(), `
				SELECT id, tenant_id, name, description, jsonb_array_length(items) as item_count, created_at, updated_at
				FROM datasets
				ORDER BY created_at DESC
			`)
			if err != nil {
				return err
			}
			defer rows.Close()

			for rows.Next() {
				var d Dataset
				if err := rows.Scan(&d.ID, &d.TenantID, &d.Name, &d.Description, &d.ItemCount, &d.CreatedAt, &d.UpdatedAt); err != nil {
					return err
				}
				datasets = append(datasets, d)
			}
			return rows.Err()
		})

		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list datasets")
			return
		}

		writeJSON(w, http.StatusOK, datasets)
	}
}

// GetDatasetHandler retrieves a single dataset by ID for the authenticated tenant.
func GetDatasetHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := TenantFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid dataset id")
			return
		}

		var dataset Dataset
		var itemsJSON []byte

		err = database.WithTenantTx(r.Context(), pool, tenantID, func(tx pgx.Tx) error {
			return tx.QueryRow(r.Context(), `
				SELECT id, tenant_id, name, description, items, created_at, updated_at
				FROM datasets
				WHERE id = $1
			`, id).Scan(
				&dataset.ID,
				&dataset.TenantID,
				&dataset.Name,
				&dataset.Description,
				&itemsJSON,
				&dataset.CreatedAt,
				&dataset.UpdatedAt,
			)
		})

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, http.StatusNotFound, "dataset not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to retrieve dataset")
			return
		}

		_ = json.Unmarshal(itemsJSON, &dataset.Items)
		dataset.ItemCount = len(dataset.Items)

		writeJSON(w, http.StatusOK, dataset)
	}
}

// UpdateDatasetHandler updates an existing dataset.
func UpdateDatasetHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := TenantFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid dataset id")
			return
		}

		var req UpdateDatasetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		var dataset Dataset
		var itemsJSON []byte

		err = database.WithTenantTx(r.Context(), pool, tenantID, func(tx pgx.Tx) error {
			// Query existing dataset
			var existingName, existingDesc string
			var existingItems []byte
			err := tx.QueryRow(r.Context(), `
				SELECT name, description, items
				FROM datasets
				WHERE id = $1
			`, id).Scan(&existingName, &existingDesc, &existingItems)
			if err != nil {
				return err
			}

			newName := existingName
			if req.Name != nil {
				trimmed := strings.TrimSpace(*req.Name)
				if trimmed == "" {
					return fmt.Errorf("dataset name cannot be empty")
				}
				newName = trimmed
			}

			newDesc := existingDesc
			if req.Description != nil {
				newDesc = *req.Description
			}

			newItems := existingItems
			if req.Items != nil {
				if err := validateDatasetItems(*req.Items); err != nil {
					return err
				}
				raw, err := json.Marshal(*req.Items)
				if err != nil {
					return err
				}
				newItems = raw
			}

			return tx.QueryRow(r.Context(), `
				UPDATE datasets
				SET name = $2, description = $3, items = $4, updated_at = NOW()
				WHERE id = $1
				RETURNING id, tenant_id, name, description, items, created_at, updated_at
			`, id, newName, newDesc, newItems).Scan(
				&dataset.ID,
				&dataset.TenantID,
				&dataset.Name,
				&dataset.Description,
				&itemsJSON,
				&dataset.CreatedAt,
				&dataset.UpdatedAt,
			)
		})

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, http.StatusNotFound, "dataset not found")
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		_ = json.Unmarshal(itemsJSON, &dataset.Items)
		dataset.ItemCount = len(dataset.Items)

		writeJSON(w, http.StatusOK, dataset)
	}
}

// DeleteDatasetHandler removes a dataset by ID.
func DeleteDatasetHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := TenantFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid dataset id")
			return
		}

		var rowsAffected int64
		err = database.WithTenantTx(r.Context(), pool, tenantID, func(tx pgx.Tx) error {
			tag, err := tx.Exec(r.Context(), `
				DELETE FROM datasets
				WHERE id = $1
			`, id)
			if err != nil {
				return err
			}
			rowsAffected = tag.RowsAffected()
			return nil
		})

		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to delete dataset")
			return
		}

		if rowsAffected == 0 {
			writeError(w, http.StatusNotFound, "dataset not found")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}
