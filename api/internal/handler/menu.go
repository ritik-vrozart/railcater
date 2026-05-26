package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/repository"
)

type portionRequest struct {
	Portion       string `json:"portion"`
	Label         string `json:"label"`
	PriceCents    int64  `json:"price_cents"`
	StockQuantity int    `json:"stock_quantity"`
	IsActive      *bool  `json:"is_active"`
	SortOrder     int    `json:"sort_order"`
}

func portionInputsFromRequest(req []portionRequest) []repository.PortionInput {
	var out []repository.PortionInput
	for _, p := range req {
		active := true
		if p.IsActive != nil {
			active = *p.IsActive
		}
		out = append(out, repository.PortionInput{
			Portion:       strings.ToLower(p.Portion),
			Label:         p.Label,
			PriceCents:    p.PriceCents,
			StockQuantity: p.StockQuantity,
			IsActive:      active,
			SortOrder:     p.SortOrder,
		})
	}
	return out
}

func (s *Server) ListVendorMenu(w http.ResponseWriter, r *http.Request) {
	vendorID, err := uuid.Parse(chi.URLParam(r, "vendorId"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid vendor id"))
		return
	}
	if err := s.ensureVendorExists(r, vendorID); err != nil {
		writeError(w, err)
		return
	}
	activeOnly := r.URL.Query().Get("active_only") != "false"
	items, err := s.menu.ListByVendor(r.Context(), vendorID, activeOnly)
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}

	if dateStr := r.URL.Query().Get("date"); dateStr != "" {
		menuDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			writeError(w, apperror.BadRequest("date must be YYYY-MM-DD"))
			return
		}
		allowed, err := s.dailyMenus.AvailableMenuItemIDs(r.Context(), vendorID, menuDate)
		if err != nil {
			writeError(w, apperror.Internal(err))
			return
		}
		if len(allowed) > 0 {
			filtered := items[:0]
			for _, it := range items {
				if allowed[it.ID] {
					filtered = append(filtered, it)
				}
			}
			items = filtered
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) ListMenuCategories(w http.ResponseWriter, r *http.Request) {
	vendorID, err := uuid.Parse(chi.URLParam(r, "vendorId"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid vendor id"))
		return
	}
	if err := s.ensureVendorExists(r, vendorID); err != nil {
		writeError(w, err)
		return
	}
	activeOnly := r.URL.Query().Get("active_only") == "true"
	items, err := s.menu.ListCategories(r.Context(), vendorID, activeOnly)
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

type createCategoryRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	FoodType    string  `json:"food_type"`
	SortOrder   int     `json:"sort_order"`
}

func (s *Server) CreateMenuCategory(w http.ResponseWriter, r *http.Request) {
	vendorID, err := uuid.Parse(chi.URLParam(r, "vendorId"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid vendor id"))
		return
	}
	if err := s.authorizeVendor(r, vendorID); err != nil {
		writeError(w, err)
		return
	}

	var req createCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, apperror.BadRequest("name is required"))
		return
	}

	c, err := s.menu.CreateCategory(r.Context(), repository.CreateCategoryInput{
		VendorID:    vendorID,
		Name:        strings.TrimSpace(req.Name),
		Description: req.Description,
		FoodType:    req.FoodType,
		SortOrder:   req.SortOrder,
	})
	if errors.Is(err, apperror.ErrConflict) {
		writeError(w, apperror.Conflict("category already exists"))
		return
	} else if err != nil {
		writeHandlerErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

type updateCategoryRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	FoodType    *string `json:"food_type"`
	SortOrder   *int    `json:"sort_order"`
	IsActive    *bool   `json:"is_active"`
}

func (s *Server) UpdateMenuCategory(w http.ResponseWriter, r *http.Request) {
	vendorID, err := uuid.Parse(chi.URLParam(r, "vendorId"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid vendor id"))
		return
	}
	categoryID, err := uuid.Parse(chi.URLParam(r, "categoryId"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid category id"))
		return
	}
	if err := s.authorizeVendor(r, vendorID); err != nil {
		writeError(w, err)
		return
	}

	var req updateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}

	c, err := s.menu.UpdateCategory(r.Context(), vendorID, categoryID, repository.UpdateCategoryInput{
		Name:        req.Name,
		Description: req.Description,
		FoodType:    req.FoodType,
		SortOrder:   req.SortOrder,
		IsActive:    req.IsActive,
	})
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("category not found"))
		return
	} else if err != nil {
		writeHandlerErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

type createMenuItemRequest struct {
	CategoryID  *uuid.UUID        `json:"category_id"`
	Name        string            `json:"name"`
	Description *string           `json:"description"`
	ImageURL    *string           `json:"image_url"`
	IsVeg       bool              `json:"is_veg"`
	IsActive    *bool             `json:"is_active"`
	Portions    []portionRequest  `json:"portions"`
}

func (s *Server) CreateMenuItem(w http.ResponseWriter, r *http.Request) {
	vendorID, err := uuid.Parse(chi.URLParam(r, "vendorId"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid vendor id"))
		return
	}
	if err := s.authorizeVendor(r, vendorID); err != nil {
		writeError(w, err)
		return
	}

	var req createMenuItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, apperror.BadRequest("name is required"))
		return
	}
	if req.CategoryID == nil {
		writeError(w, apperror.BadRequest("category_id is required"))
		return
	}

	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}

	item, err := s.menu.CreateItem(r.Context(), repository.CreateMenuItemInput{
		VendorID:    vendorID,
		CategoryID:  req.CategoryID,
		Name:        strings.TrimSpace(req.Name),
		Description: req.Description,
		ImageURL:    req.ImageURL,
		IsVeg:       req.IsVeg,
		IsActive:    active,
		Portions:    portionInputsFromRequest(req.Portions),
	})
	if err != nil {
		writeHandlerErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

type updateMenuItemRequest struct {
	CategoryID  *uuid.UUID       `json:"category_id"`
	Name        *string          `json:"name"`
	Description *string          `json:"description"`
	ImageURL    *string          `json:"image_url"`
	IsVeg       *bool            `json:"is_veg"`
	IsActive    *bool            `json:"is_active"`
	Portions    []portionRequest `json:"portions"`
}

func (s *Server) UpdateMenuItem(w http.ResponseWriter, r *http.Request) {
	vendorID, err := uuid.Parse(chi.URLParam(r, "vendorId"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid vendor id"))
		return
	}
	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid menu item id"))
		return
	}
	if err := s.authorizeVendor(r, vendorID); err != nil {
		writeError(w, err)
		return
	}

	var req updateMenuItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}

	item, err := s.menu.UpdateItem(r.Context(), vendorID, itemID, repository.UpdateMenuItemInput{
		CategoryID:  req.CategoryID,
		Name:        req.Name,
		Description: req.Description,
		ImageURL:    req.ImageURL,
		IsVeg:       req.IsVeg,
		IsActive:    req.IsActive,
		Portions:    portionInputsFromRequest(req.Portions),
	})
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("menu item not found"))
		return
	} else if err != nil {
		writeHandlerErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) ensureVendorExists(r *http.Request, vendorID uuid.UUID) error {
	_, err := s.vendors.GetByID(r.Context(), s.tenantID, vendorID)
	if errors.Is(err, apperror.ErrNotFound) {
		return apperror.NotFound("vendor not found")
	}
	if err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func writeHandlerErr(w http.ResponseWriter, err error) {
	var httpErr *apperror.HTTPError
	if errors.As(err, &httpErr) {
		writeError(w, err)
		return
	}
	writeError(w, apperror.Internal(err))
}
