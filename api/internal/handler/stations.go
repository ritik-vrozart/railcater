package handler

import (
	"net/http"

	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
)

func (s *Server) ListStations(w http.ResponseWriter, r *http.Request) {
	items, err := s.stations.List(r.Context())
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}
