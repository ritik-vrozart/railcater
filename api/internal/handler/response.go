package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
)

type errorBody struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	var httpErr *apperror.HTTPError
	if errors.As(err, &httpErr) {
		msg := httpErr.Message
		if msg == "" && httpErr.Err != nil {
			msg = httpErr.Err.Error()
		}
		writeJSON(w, httpErr.Status, errorBody{Error: msg})
		return
	}

	slog.Error("unhandled error", "err", err)
	writeJSON(w, http.StatusInternalServerError, errorBody{Error: "internal server error"})
}
