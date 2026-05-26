package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/notify"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/repository"
)

func (s *Server) ListOrders(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	status := r.URL.Query().Get("status")

	var vendorID *uuid.UUID
	if v := r.URL.Query().Get("vendor_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeError(w, apperror.BadRequest("invalid vendor_id"))
			return
		}
		vendorID = &id
	}

	var customerID *uuid.UUID
	if v := r.URL.Query().Get("customer_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeError(w, apperror.BadRequest("invalid customer_id"))
			return
		}
		customerID = &id
	}

	items, total, err := s.orders.List(r.Context(), s.tenantID, page, perPage, repository.OrderListFilter{
		Status:     status,
		VendorID:   vendorID,
		CustomerID: customerID,
	})
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": paginateMeta(page, perPage, total),
	})
}

func (s *Server) GetOrder(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid order id"))
		return
	}

	o, err := s.orders.GetByID(r.Context(), s.tenantID, id)
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("order not found"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusOK, o)
}

type orderLineRequest struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}

type createOrderRequest struct {
	CustomerID *uuid.UUID         `json:"customer_id"`
	Source     string             `json:"source"`
	Notes      *string            `json:"notes"`
	Items      []orderLineRequest `json:"items"`
}

func (s *Server) CreateOrder(w http.ResponseWriter, r *http.Request) {
	s.createOrder(w, r, "dashboard")
}

func (s *Server) CreateWhatsAppOrder(w http.ResponseWriter, r *http.Request) {
	s.createOrder(w, r, "whatsapp")
}

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request, defaultSource string) {
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}
	if req.Source == "" {
		req.Source = defaultSource
	}

	var lines []repository.OrderLineInput
	for _, item := range req.Items {
		lines = append(lines, repository.OrderLineInput{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	o, err := s.orders.Create(r.Context(), repository.CreateOrderInput{
		TenantID:   s.tenantID,
		CustomerID: req.CustomerID,
		Source:     req.Source,
		Notes:      req.Notes,
		Items:      lines,
	})
	var httpErr *apperror.HTTPError
	if errors.As(err, &httpErr) {
		writeError(w, err)
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	writeJSON(w, http.StatusCreated, o)
}

type updateOrderStatusRequest struct {
	Status string `json:"status"`
}

var validOrderStatuses = map[string]bool{
	"pending": true, "confirmed": true, "preparing": true, "ready": true,
	"dispatched": true, "processing": true, "shipped": true,
	"delivered": true, "cancelled": true,
}

// trainOrderFlow defines allowed status transitions for pantry train orders.
var trainOrderTransitions = map[string][]string{
	"confirmed":  {"preparing", "cancelled"},
	"preparing":  {"ready", "cancelled"},
	"ready":      {"dispatched", "cancelled"},
	"dispatched": {"delivered", "cancelled"},
	"pending":    {"confirmed", "cancelled"},
	"processing": {"shipped", "delivered", "cancelled"},
	"shipped":    {"delivered", "cancelled"},
}

func (s *Server) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, apperror.BadRequest("invalid order id"))
		return
	}
	if err := s.requireAuth(r); err != nil {
		writeError(w, err)
		return
	}

	var req updateOrderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}
	if !validOrderStatuses[req.Status] {
		writeError(w, apperror.BadRequest("invalid order status"))
		return
	}

	existing, err := s.orders.GetByID(r.Context(), s.tenantID, id)
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("order not found"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	if err := s.authorizeOrder(r, existing.VendorID); err != nil {
		writeError(w, err)
		return
	}
	if existing.Source == "train" {
		if !canTransitionTrainOrder(existing.Status, req.Status) {
			writeError(w, apperror.BadRequest("invalid status transition from "+existing.Status+" to "+req.Status))
			return
		}
	}

	o, err := s.orders.UpdateStatus(r.Context(), s.tenantID, id, req.Status)
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("order not found"))
		return
	} else if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}

	if existing.Status != req.Status && existing.Source == "train" {
		orderCopy := o
		go notify.OrderStatus(context.Background(), s.agentNotifyURL, s.agentNotifySecret, orderCopy, req.Status)
	}

	writeJSON(w, http.StatusOK, o)
}

func canTransitionTrainOrder(from, to string) bool {
	if from == to {
		return true
	}
	allowed, ok := trainOrderTransitions[from]
	if !ok {
		return to == "cancelled"
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// CheckStock is used by the FastAPI agent before confirming an order.
type checkStockRequest struct {
	Items []orderLineRequest `json:"items"`
}

type checkStockItem struct {
	ProductID uuid.UUID `json:"product_id"`
	SKU       string    `json:"sku"`
	Requested int       `json:"requested"`
	Available int       `json:"available"`
	OK        bool      `json:"ok"`
}

func (s *Server) CheckStock(w http.ResponseWriter, r *http.Request) {
	var req checkStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}

	var results []checkStockItem
	allOK := true
	for _, item := range req.Items {
		p, err := s.products.GetByID(r.Context(), s.tenantID, item.ProductID)
		if errors.Is(err, apperror.ErrNotFound) {
			writeError(w, apperror.BadRequest("product not found"))
			return
		} else if err != nil {
			writeError(w, apperror.Internal(err))
			return
		}
		available := p.Quantity - p.Reserved
		ok := available >= item.Quantity
		if !ok {
			allOK = false
		}
		results = append(results, checkStockItem{
			ProductID: item.ProductID,
			SKU:       p.SKU,
			Requested: item.Quantity,
			Available: available,
			OK:        ok,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":    allOK,
		"items": results,
	})
}
