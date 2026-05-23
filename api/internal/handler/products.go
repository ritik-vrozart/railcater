package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/repository"
)

func (s *Server) ListProducts(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	activeOnly := r.URL.Query().Get("active_only") == "true"

	items, total, err := s.products.List(r.Context(), s.tenantID, page, perPage, activeOnly)
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": paginateMeta(page, perPage, total),
	})
}

func (s *Server) GetProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid product id"))
		return
	}

	p, err := s.products.GetByID(r.Context(), s.tenantID, id)
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("product not found"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, p)
}

type createProductRequest struct {
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Unit        string  `json:"unit"`
	PriceCents  int64   `json:"price_cents"`
	Quantity    int     `json:"quantity"`
}

func (s *Server) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req createProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}
	if req.SKU == "" || req.Name == "" {
		writeError(w, apperror.BadRequest("sku and name are required"))
		return
	}
	if req.Unit == "" {
		req.Unit = "pcs"
	}
	if req.PriceCents < 0 {
		writeError(w, apperror.BadRequest("price_cents must be non-negative"))
		return
	}

	p, err := s.products.Create(r.Context(), repository.CreateProductInput{
		TenantID:    s.tenantID,
		SKU:         req.SKU,
		Name:        req.Name,
		Description: req.Description,
		Unit:        req.Unit,
		PriceCents:  req.PriceCents,
		Quantity:    req.Quantity,
	})
	if errors.Is(err, apperror.ErrConflict) {
		writeError(w, apperror.Conflict("product sku already exists"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

type updateProductRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Unit        *string `json:"unit"`
	PriceCents  *int64  `json:"price_cents"`
	IsActive    *bool   `json:"is_active"`
}

func (s *Server) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid product id"))
		return
	}

	var req updateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}

	p, err := s.products.Update(r.Context(), s.tenantID, id, repository.UpdateProductInput{
		Name:        req.Name,
		Description: req.Description,
		Unit:        req.Unit,
		PriceCents:  req.PriceCents,
		IsActive:    req.IsActive,
	})
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("product not found"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func paginateMeta(page, perPage, total int) map[string]int {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	return map[string]int{"page": page, "per_page": perPage, "total": total}
}
