package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
)

func (s *Server) ListTrains(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active_only") == "true"
	items, err := s.trains.List(r.Context(), s.tenantID, activeOnly)
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) GetTrain(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid train id"))
		return
	}

	runDate := r.URL.Query().Get("run_date")
	detail, err := s.trains.GetByID(r.Context(), s.tenantID, id)
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("train not found"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}

	if runDate != "" {
		d, err := time.Parse("2006-01-02", runDate)
		if err != nil {
			writeError(w, apperror.BadRequest("invalid run_date, use YYYY-MM-DD"))
			return
		}
		run, err := s.trains.GetRun(r.Context(), id, d)
		if err == nil {
			detail.Run = &run
		} else if !errors.Is(err, apperror.ErrNotFound) {
			writeError(w, apperror.Internal(err))
			return
		}
	}

	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) GetTrainStations(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid train id"))
		return
	}

	detail, err := s.trains.GetByID(r.Context(), s.tenantID, id)
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("train not found"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": detail.Stops})
}

type updateTrainDelayRequest struct {
	RunDate       string `json:"run_date"`
	DelayMinutes  int    `json:"delay_minutes"`
}

func (s *Server) UpdateTrainDelay(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid train id"))
		return
	}

	var req updateTrainDelayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}
	if req.RunDate == "" {
		writeError(w, apperror.BadRequest("run_date is required"))
		return
	}

	runDate, err := time.Parse("2006-01-02", req.RunDate)
	if err != nil {
		writeError(w, apperror.BadRequest("invalid run_date, use YYYY-MM-DD"))
		return
	}

	if _, err := s.trains.GetByID(r.Context(), s.tenantID, id); errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("train not found"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}

	run, err := s.trains.UpsertRunDelay(r.Context(), id, runDate, req.DelayMinutes)
	if err != nil {
		var httpErr *apperror.HTTPError
		if errors.As(err, &httpErr) {
			writeError(w, err)
			return
		}
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, run)
}
