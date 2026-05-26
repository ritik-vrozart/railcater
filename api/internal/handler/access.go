package handler

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/auth"
	appmw "github.com/ritikkumarpathak/whatsapp-bot/api/internal/middleware"
)

func (s *Server) claimsFromRequest(r *http.Request) (auth.Claims, bool) {
	return appmw.ClaimsFromContext(r.Context())
}

func (s *Server) requireAuth(r *http.Request) error {
	if _, ok := s.claimsFromRequest(r); !ok {
		return apperror.Unauthorized("authentication required")
	}
	return nil
}

func (s *Server) authorizeVendor(r *http.Request, vendorID uuid.UUID) error {
	if err := s.requireAuth(r); err != nil {
		return err
	}
	claims, _ := s.claimsFromRequest(r)
	if claims.Role == "super_admin" {
		return nil
	}
	if claims.Role != "vendor_admin" {
		return apperror.Forbidden("pantry access only")
	}
	u, err := s.users.GetByID(r.Context(), claims.TenantID, claims.UserID)
	if errors.Is(err, apperror.ErrNotFound) {
		return apperror.Unauthorized("user not found")
	} else if err != nil {
		return apperror.Internal(err)
	}
	if u.VendorID == nil || *u.VendorID != vendorID {
		return apperror.Forbidden("not your pantry")
	}
	return nil
}

func (s *Server) authorizeOrder(r *http.Request, orderVendorID *uuid.UUID) error {
	if err := s.requireAuth(r); err != nil {
		return err
	}
	claims, _ := s.claimsFromRequest(r)
	if claims.Role == "super_admin" {
		return nil
	}
	if claims.Role != "vendor_admin" {
		return apperror.Forbidden("pantry access only")
	}
	if orderVendorID == nil {
		return apperror.Forbidden("order has no pantry")
	}
	u, err := s.users.GetByID(r.Context(), claims.TenantID, claims.UserID)
	if errors.Is(err, apperror.ErrNotFound) {
		return apperror.Unauthorized("user not found")
	} else if err != nil {
		return apperror.Internal(err)
	}
	if u.VendorID == nil || *u.VendorID != *orderVendorID {
		return apperror.Forbidden("not your order")
	}
	return nil
}
