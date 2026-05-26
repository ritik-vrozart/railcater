package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/models"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/repository"
)

type trainOrderLineRequest struct {
	MenuPortionID uuid.UUID `json:"menu_portion_id"`
	Quantity      int       `json:"quantity"`
}

type createWhatsAppTrainOrderRequest struct {
	TrainNumber   string                  `json:"train_number"`
	TrainID       *uuid.UUID              `json:"train_id"`
	VendorID      *uuid.UUID              `json:"vendor_id"`
	CustomerID    *uuid.UUID              `json:"customer_id"`
	PassengerName string                  `json:"passenger_name"`
	Coach         string                  `json:"coach"`
	Berth         string                  `json:"berth"`
	Notes         *string                 `json:"notes"`
	Items         []trainOrderLineRequest `json:"items"`
}

func (s *Server) CreateWhatsAppTrainOrder(w http.ResponseWriter, r *http.Request) {
	var req createWhatsAppTrainOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}
	trainNumber := strings.TrimSpace(req.TrainNumber)
	passengerName := strings.TrimSpace(req.PassengerName)
	coach := strings.TrimSpace(strings.ToUpper(req.Coach))
	berth := strings.TrimSpace(req.Berth)

	if trainNumber == "" && (req.TrainID == nil || *req.TrainID == uuid.Nil) {
		writeError(w, apperror.BadRequest("train_number or train_id is required"))
		return
	}
	if passengerName == "" {
		writeError(w, apperror.BadRequest("passenger_name is required"))
		return
	}
	if coach == "" || berth == "" {
		writeError(w, apperror.BadRequest("coach and berth are required"))
		return
	}
	if len(req.Items) == 0 {
		writeError(w, apperror.BadRequest("items are required"))
		return
	}

	var train models.Train
	var err error
	if req.TrainID != nil && *req.TrainID != uuid.Nil {
		detail, derr := s.trains.GetByID(r.Context(), s.tenantID, *req.TrainID)
		if errors.Is(derr, apperror.ErrNotFound) {
			writeError(w, apperror.NotFound("train not found"))
			return
		} else if derr != nil {
			writeError(w, apperror.Internal(derr))
			return
		}
		train = detail.Train
	} else {
		train, err = s.trains.GetByNumber(r.Context(), s.tenantID, trainNumber)
		if errors.Is(err, apperror.ErrNotFound) {
			writeError(w, apperror.NotFound("train not found"))
			return
		} else if err != nil {
			writeError(w, apperror.Internal(err))
			return
		}
	}

	pantries, err := s.vendors.ListForTrain(r.Context(), s.tenantID, train.ID)
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	if len(pantries) == 0 {
		writeError(w, apperror.NotFound("no pantry on this train"))
		return
	}

	vendorID := uuid.Nil
	if req.VendorID != nil && *req.VendorID != uuid.Nil {
		vendorID = *req.VendorID
		found := false
		for _, p := range pantries {
			if p.ID == vendorID {
				found = true
				break
			}
		}
		if !found {
			writeError(w, apperror.BadRequest("pantry does not serve this train"))
			return
		}
	} else if len(pantries) == 1 {
		vendorID = pantries[0].ID
	} else {
		writeError(w, apperror.BadRequest("multiple pantries on this train; vendor_id is required"))
		return
	}

	// Onboard delivery to coach/seat — no station selection.
	dw := onboardDeliveryWindow()

	var lines []repository.TrainOrderLineInput
	for _, item := range req.Items {
		lines = append(lines, repository.TrainOrderLineInput{
			MenuPortionID: item.MenuPortionID,
			Quantity:      item.Quantity,
		})
	}

	notes := req.Notes
	if notes == nil {
		n := fmt.Sprintf("WhatsApp order · Train %s · %s", train.Number, passengerName)
		notes = &n
	}

	pnrPlaceholder := fmt.Sprintf("WA-%s", train.Number)

	o, err := s.orders.CreateTrain(r.Context(), repository.CreateTrainOrderInput{
		TenantID:            s.tenantID,
		PNR:                 pnrPlaceholder,
		TrainID:             train.ID,
		StationID:           nil,
		VendorID:            vendorID,
		CustomerID:          req.CustomerID,
		Coach:               coach,
		Berth:               berth,
		PassengerName:       passengerName,
		DeliveryWindowStart: dw.start,
		DeliveryWindowEnd:   dw.end,
		Notes:               notes,
		Items:               lines,
	})
	if err != nil {
		writeTrainOrderError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, o)
}

type onboardWindow struct {
	start time.Time
	end   time.Time
}

func onboardDeliveryWindow() onboardWindow {
	now := time.Now()
	return onboardWindow{
		start: now.Add(30 * time.Minute),
		end:   now.Add(3 * time.Hour),
	}
}

func writeTrainOrderError(w http.ResponseWriter, err error) {
	var httpErr *apperror.HTTPError
	if errors.As(err, &httpErr) {
		writeError(w, err)
		return
	}
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("resource not found"))
		return
	}
	writeError(w, apperror.Internal(err))
}
