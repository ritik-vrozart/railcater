package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
)

func (s *Server) LookupPNR(w http.ResponseWriter, r *http.Request) {
	pnr := chi.URLParam(r, "pnr")
	lookup, err := s.pnr.Lookup(r.Context(), s.tenantID, pnr)
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("PNR not found"))
		return
	}
	var httpErr *apperror.HTTPError
	if errors.As(err, &httpErr) {
		writeError(w, err)
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, lookup)
}
