package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/auth"
	appmw "github.com/ritikkumarpathak/whatsapp-bot/api/internal/middleware"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/models"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/repository"
)

var defaultVendorID = uuid.MustParse("c3000001-0000-4000-8000-000000000001")

type registerRequest struct {
	Name     string  `json:"name"`
	Email    string  `json:"email"`
	Phone    *string `json:"phone"`
	Password string  `json:"password"`
	Role     string  `json:"role"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}

	if err := validateRegister(req); err != nil {
		writeError(w, err)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}

	role := req.Role
	if role == "" {
		role = "passenger"
	}

	var vendorID *uuid.UUID
	if role == "vendor_admin" {
		vendorID = &defaultVendorID
	}

	u, err := s.users.Create(r.Context(), repository.CreateUserInput{
		TenantID:     s.tenantID,
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: hash,
		Role:         role,
		VendorID:     vendorID,
	})
	if errors.Is(err, apperror.ErrConflict) {
		writeError(w, apperror.Conflict("email already registered"))
		return
	} else if err != nil {
		var httpErr *apperror.HTTPError
		if errors.As(err, &httpErr) {
			writeError(w, err)
			return
		}
		writeError(w, apperror.Internal(err))
		return
	}

	token, err := auth.IssueToken(s.jwtSecret, u.ID, u.TenantID, u.Email, u.Role, s.jwtExpiry)
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}

	writeJSON(w, http.StatusCreated, models.AuthResponse{Token: token, User: u})
}

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || req.Password == "" {
		writeError(w, apperror.BadRequest("email and password are required"))
		return
	}

	u, hash, err := s.users.GetByEmail(r.Context(), s.tenantID, email)
	if errors.Is(err, apperror.ErrNotFound) || !auth.CheckPassword(hash, req.Password) {
		writeError(w, apperror.Unauthorized("invalid email or password"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}

	if !u.IsActive {
		writeError(w, apperror.Unauthorized("account is disabled"))
		return
	}

	token, err := auth.IssueToken(s.jwtSecret, u.ID, u.TenantID, u.Email, u.Role, s.jwtExpiry)
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}

	writeJSON(w, http.StatusOK, models.AuthResponse{Token: token, User: u})
}

func (s *Server) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := appmw.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, apperror.Unauthorized("unauthorized"))
		return
	}

	u, err := s.users.GetByID(r.Context(), claims.TenantID, claims.UserID)
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.Unauthorized("user not found"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}

	writeJSON(w, http.StatusOK, u)
}

func validateRegister(req registerRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return apperror.BadRequest("name is required")
	}
	if !strings.Contains(req.Email, "@") {
		return apperror.BadRequest("valid email is required")
	}
	if len(req.Password) < 8 {
		return apperror.BadRequest("password must be at least 8 characters")
	}
	if !passwordStrongEnough(req.Password) {
		return apperror.BadRequest("password must include a letter and a number")
	}
	return nil
}

func passwordStrongEnough(p string) bool {
	var hasLetter, hasDigit bool
	for _, c := range p {
		if unicode.IsLetter(c) {
			hasLetter = true
		}
		if unicode.IsDigit(c) {
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}
