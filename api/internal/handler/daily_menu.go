package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/repository"
)

type dailyMenuItemRequest struct {
	MenuItemID    uuid.UUID `json:"menu_item_id"`
	IsAvailable   bool      `json:"is_available"`
	StockOverride *int      `json:"stock_override"`
}

type setDailyMenuRequest struct {
	Items []dailyMenuItemRequest `json:"items"`
	Notes *string                `json:"notes"`
}

func parseMenuDate(r *http.Request) (time.Time, error) {
	raw := r.URL.Query().Get("date")
	if raw == "" {
		return time.Now().UTC().Truncate(24 * time.Hour), nil
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, apperror.BadRequest("date must be YYYY-MM-DD")
	}
	return t, nil
}

func (s *Server) GetDailyMenu(w http.ResponseWriter, r *http.Request) {
	vendorID, err := uuid.Parse(chi.URLParam(r, "vendorId"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid vendor id"))
		return
	}
	if err := s.authorizeVendor(r, vendorID); err != nil {
		writeError(w, err)
		return
	}

	menuDate, err := parseMenuDate(r)
	if err != nil {
		writeError(w, err)
		return
	}

	m, err := s.dailyMenus.GetOrCreate(r.Context(), vendorID, menuDate)
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (s *Server) SetDailyMenu(w http.ResponseWriter, r *http.Request) {
	vendorID, err := uuid.Parse(chi.URLParam(r, "vendorId"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid vendor id"))
		return
	}
	if err := s.authorizeVendor(r, vendorID); err != nil {
		writeError(w, err)
		return
	}

	menuDate, err := parseMenuDate(r)
	if err != nil {
		writeError(w, err)
		return
	}

	var req setDailyMenuRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}

	m, err := s.dailyMenus.GetOrCreate(r.Context(), vendorID, menuDate)
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}

	var items []repository.DailyMenuItemInput
	for _, it := range req.Items {
		items = append(items, repository.DailyMenuItemInput{
			MenuItemID:    it.MenuItemID,
			IsAvailable:   it.IsAvailable,
			StockOverride: it.StockOverride,
		})
	}

	updated, err := s.dailyMenus.SetItems(r.Context(), m.ID, items)
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("daily menu not found"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
