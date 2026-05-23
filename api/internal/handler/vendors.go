package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
)

func (s *Server) ListVendors(w http.ResponseWriter, r *http.Request) {
	approvedOnly := r.URL.Query().Get("approved_only") != "false"
	items, err := s.vendors.List(r.Context(), s.tenantID, approvedOnly)
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) GetVendor(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid vendor id"))
		return
	}

	v, err := s.vendors.GetByID(r.Context(), s.tenantID, id)
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("vendor not found"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) ListStationVendors(w http.ResponseWriter, r *http.Request) {
	stationID, err := uuid.Parse(chi.URLParam(r, "stationId"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid station id"))
		return
	}

	items, err := s.vendors.ListAtStation(r.Context(), s.tenantID, stationID)
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}
