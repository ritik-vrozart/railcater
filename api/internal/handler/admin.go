package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/auth"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/models"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/repository"
)

type invitePantryRequest struct {
	PantryName  string  `json:"pantry_name"`
	TrainNumber string  `json:"train_number"`
	TrainName   string  `json:"train_name"`
	AdminName   string  `json:"admin_name"`
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	Phone       *string `json:"phone"`
}

var codeSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

func (s *Server) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	period, err := parseDateQuery(r)
	if err != nil {
		writeError(w, err)
		return
	}
	dash, err := s.vendors.AdminDashboard(r.Context(), s.tenantID, period)
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, dash)
}

func (s *Server) ListPantries(w http.ResponseWriter, r *http.Request) {
	period, err := parseDateQuery(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := s.vendors.ListDetails(r.Context(), s.tenantID, period)
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	if items == nil {
		items = []models.VendorDetail{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":      items,
		"date_from": period.FromISO(),
		"date_to":   period.ToISO(),
	})
}

func (s *Server) ListAdminOrders(w http.ResponseWriter, r *http.Request) {
	period, err := parseDateQuery(r)
	if err != nil {
		writeError(w, err)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 {
		perPage = 50
	}

	f := orderListFilterFromRequest(r, period)
	f.TrainOnly = true

	if v := r.URL.Query().Get("vendor_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeError(w, apperror.BadRequest("invalid vendor_id"))
			return
		}
		f.VendorID = &id
	}

	items, total, err := s.orders.List(r.Context(), s.tenantID, page, perPage, f)
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":      items,
		"meta":      paginateMeta(page, perPage, total),
		"date_from": period.FromISO(),
		"date_to":   period.ToISO(),
	})
}

func (s *Server) GetPantry(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid pantry id"))
		return
	}
	period, err := parseDateQuery(r)
	if err != nil {
		writeError(w, err)
		return
	}
	detail, err := s.vendors.GetDetail(r.Context(), s.tenantID, id, period)
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("pantry not found"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) InvitePantry(w http.ResponseWriter, r *http.Request) {
	var req invitePantryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}

	pantryName := strings.TrimSpace(req.PantryName)
	trainNumber := strings.TrimSpace(req.TrainNumber)
	trainName := strings.TrimSpace(req.TrainName)
	adminName := strings.TrimSpace(req.AdminName)
	if adminName == "" {
		adminName = pantryName
	}

	if pantryName == "" || trainNumber == "" {
		writeError(w, apperror.BadRequest("pantry_name and train_number are required"))
		return
	}
	if !strings.Contains(req.Email, "@") {
		writeError(w, apperror.BadRequest("valid email is required"))
		return
	}
	if len(req.Password) < 8 {
		writeError(w, apperror.BadRequest("password must be at least 8 characters"))
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}

	train, err := s.trains.UpsertByNumber(r.Context(), s.tenantID, trainNumber, trainName)
	if err != nil {
		writeHandlerErr(w, err)
		return
	}

	code := strings.ToUpper(codeSanitizer.ReplaceAllString(strings.ToLower(pantryName), "-"))
	code = strings.Trim(code, "-")
	if code == "" {
		code = "PANTRY-" + trainNumber
	}
	if len(code) > 40 {
		code = code[:40]
	}

	vendor, err := s.vendors.Create(r.Context(), repository.CreateVendorInput{
		TenantID: s.tenantID,
		Name:     pantryName,
		Code:     code,
		Phone:    req.Phone,
	})
	if errors.Is(err, apperror.ErrConflict) {
		writeError(w, apperror.Conflict("pantry code already exists; use a different name"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}

	if err := s.vendors.LinkTrain(r.Context(), vendor.ID, train.ID); err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	_ = s.vendors.LinkStationsFromTrain(r.Context(), vendor.ID, train.ID)

	u, err := s.users.Create(r.Context(), repository.CreateUserInput{
		TenantID:     s.tenantID,
		Name:         adminName,
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: hash,
		Role:         "vendor_admin",
		VendorID:     &vendor.ID,
	})
	if errors.Is(err, apperror.ErrConflict) {
		writeError(w, apperror.Conflict("email already registered"))
		return
	} else if err != nil {
		writeHandlerErr(w, err)
		return
	}

	detail, err := s.vendors.GetDetail(r.Context(), s.tenantID, vendor.ID, defaultAdminPeriod())
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"pantry": detail,
		"admin":  u,
	})
}
