package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
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

type createTrainOrderRequest struct {
	PNR        string                  `json:"pnr"`
	StationID  uuid.UUID               `json:"station_id"`
	VendorID   uuid.UUID               `json:"vendor_id"`
	CustomerID *uuid.UUID              `json:"customer_id"`
	Coach      string                  `json:"coach"`
	Berth      string                  `json:"berth"`
	Notes      *string                 `json:"notes"`
	Items      []trainOrderLineRequest `json:"items"`
}

type validateDeliveryRequest struct {
	PNR       string    `json:"pnr"`
	StationID uuid.UUID `json:"station_id"`
}

func (s *Server) ValidateDelivery(w http.ResponseWriter, r *http.Request) {
	var req validateDeliveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}
	if req.PNR == "" || req.StationID == uuid.Nil {
		writeError(w, apperror.BadRequest("pnr and station_id are required"))
		return
	}

	lookup, dw, err := s.validateTrainDelivery(r.Context(), req.PNR, req.StationID)
	if err != nil {
		writeTrainOrderError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"pnr":             lookup.PNR,
		"passenger_name":  lookup.PassengerName,
		"train":           lookup.Train,
		"delivery_window": dw,
	})
}

func (s *Server) CreateTrainOrder(w http.ResponseWriter, r *http.Request) {
	var req createTrainOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, apperror.BadRequest("invalid json body"))
		return
	}
	if req.PNR == "" || req.StationID == uuid.Nil || req.VendorID == uuid.Nil {
		writeError(w, apperror.BadRequest("pnr, station_id, and vendor_id are required"))
		return
	}
	if len(req.Items) == 0 {
		writeError(w, apperror.BadRequest("items are required"))
		return
	}

	lookup, dw, err := s.validateTrainDelivery(r.Context(), req.PNR, req.StationID)
	if err != nil {
		writeTrainOrderError(w, err)
		return
	}
	if !dw.Feasible {
		writeError(w, apperror.Unprocessable(dw.FeasibilityMessage))
		return
	}

	serves, err := s.vendors.ServesStation(r.Context(), req.VendorID, req.StationID)
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	if !serves {
		writeError(w, apperror.BadRequest("vendor does not serve this station"))
		return
	}

	coach := req.Coach
	berth := req.Berth
	if coach == "" {
		coach = lookup.Coach
	}
	if berth == "" {
		berth = lookup.Berth
	}

	var lines []repository.TrainOrderLineInput
	for _, item := range req.Items {
		lines = append(lines, repository.TrainOrderLineInput{
			MenuPortionID: item.MenuPortionID,
			Quantity:      item.Quantity,
		})
	}

	o, err := s.orders.CreateTrain(r.Context(), repository.CreateTrainOrderInput{
		TenantID:            s.tenantID,
		PNR:                 lookup.PNR,
		TrainID:             lookup.Train.ID,
		StationID:           req.StationID,
		VendorID:            req.VendorID,
		CustomerID:          req.CustomerID,
		Coach:               coach,
		Berth:               berth,
		PassengerName:       lookup.PassengerName,
		DeliveryWindowStart: dw.DeliveryWindowStart,
		DeliveryWindowEnd:   dw.DeliveryWindowEnd,
		Notes:               req.Notes,
		Items:               lines,
	})
	if err != nil {
		writeTrainOrderError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, o)
}

func (s *Server) validateTrainDelivery(ctx context.Context, pnr string, stationID uuid.UUID) (models.PNRLookup, models.DeliveryWindow, error) {
	lookup, err := s.pnr.Lookup(ctx, s.tenantID, pnr)
	if err != nil {
		return models.PNRLookup{}, models.DeliveryWindow{}, err
	}

	onJourney, err := s.trains.IsStopOnJourney(ctx, lookup.Train.ID, stationID, lookup.FromStation.ID, lookup.ToStation.ID)
	if err != nil {
		return models.PNRLookup{}, models.DeliveryWindow{}, err
	}
	if !onJourney {
		return models.PNRLookup{}, models.DeliveryWindow{}, apperror.BadRequest("station is not on your journey route")
	}

	journeyDate, err := time.Parse("2006-01-02", lookup.JourneyDate)
	if err != nil {
		return models.PNRLookup{}, models.DeliveryWindow{}, apperror.Internal(err)
	}

	dw, err := s.trains.ComputeDeliveryWindow(ctx, lookup.Train.ID, stationID, journeyDate)
	if err != nil {
		return models.PNRLookup{}, models.DeliveryWindow{}, err
	}

	return lookup, dw, nil
}

func writeTrainOrderError(w http.ResponseWriter, err error) {
	var httpErr *apperror.HTTPError
	if errors.As(err, &httpErr) {
		writeError(w, err)
		return
	}
	if errors.Is(err, apperror.ErrNotFound) {
		writeError(w, apperror.NotFound("PNR not found"))
		return
	}
	writeError(w, apperror.Internal(err))
}
