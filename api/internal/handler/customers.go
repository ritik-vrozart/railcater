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

func (s *Server) ListCustomers(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	search := r.URL.Query().Get("q")

	items, total, err := s.customers.List(r.Context(), s.tenantID, page, perPage, search)
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": paginateMeta(page, perPage, total),
	})
}

func (s *Server) GetCustomerByPhone(w http.ResponseWriter, r *http.Request) {
	phone := r.URL.Query().Get("phone")
	if phone == "" {
		writeError(w, apperror.BadRequest("phone query parameter is required"))
		return
	}

	c, err := s.customers.GetByPhone(r.Context(), s.tenantID, phone)
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("customer not found"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) GetCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid customer id"))
		return
	}

	c, err := s.customers.GetByID(r.Context(), s.tenantID, id)
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("customer not found"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, c)
}

type createCustomerRequest struct {
	Name              string  `json:"name"`
	Phone             *string `json:"phone"`
	Email             *string `json:"email"`
	PreferredLanguage string  `json:"preferred_language"`
	Address           *string `json:"address"`
}

func (s *Server) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	var req createCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}
	if req.Name == "" {
		writeError(w, apperror.BadRequest("name is required"))
		return
	}
	if req.PreferredLanguage == "" {
		req.PreferredLanguage = "en"
	}

	c, err := s.customers.Create(r.Context(), repository.CreateCustomerInput{
		TenantID:          s.tenantID,
		Name:              req.Name,
		Phone:             req.Phone,
		Email:             req.Email,
		PreferredLanguage: req.PreferredLanguage,
		Address:           req.Address,
	})
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

type updateCustomerRequest struct {
	Name              *string `json:"name"`
	Phone             *string `json:"phone"`
	Email             *string `json:"email"`
	PreferredLanguage *string `json:"preferred_language"`
	Address           *string `json:"address"`
}

func (s *Server) UpdateCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid customer id"))
		return
	}

	var req updateCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}

	c, err := s.customers.Update(r.Context(), s.tenantID, id, repository.UpdateCustomerInput{
		Name:              req.Name,
		Phone:             req.Phone,
		Email:             req.Email,
		PreferredLanguage: req.PreferredLanguage,
		Address:           req.Address,
	})
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("customer not found"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, c)
}
