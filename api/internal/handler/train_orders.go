package handler

import (
	"context"
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

type createWhatsAppTrainOrderRequest struct {
	TrainNumber    string                  `json:"train_number"`
	TrainID        *uuid.UUID              `json:"train_id"`
	VendorID       *uuid.UUID              `json:"vendor_id"`
	StationID      *uuid.UUID              `json:"station_id"`
	CustomerID     *uuid.UUID              `json:"customer_id"`
	PassengerName  string                  `json:"passenger_name"`
	Coach          string                  `json:"coach"`
	Berth          string                  `json:"berth"`
	Notes          *string                 `json:"notes"`
	Items          []trainOrderLineRequest `json:"items"`
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

	stationID := uuid.Nil
	if req.StationID != nil && *req.StationID != uuid.Nil {
		stationID = *req.StationID
	} else {
		stationID, err = s.vendors.FirstStationOnTrainRoute(r.Context(), vendorID, train.ID)
		if errors.Is(err, apperror.ErrNotFound) {
			writeError(w, apperror.BadRequest("pantry has no delivery station on this train route"))
			return
		} else if err != nil {
			writeError(w, apperror.Internal(err))
			return
		}
	}

	dw, err := s.resolveWhatsAppDelivery(r.Context(), train.ID, stationID)
	if err != nil {
		writeTrainOrderError(w, err)
		return
	}

	serves, err := s.vendors.ServesStation(r.Context(), vendorID, stationID)
	if err != nil {
		writeError(w, apperror.Internal(err))
		return
	}
	if !serves {
		writeError(w, apperror.BadRequest("pantry does not serve this station"))
		return
	}

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
		StationID:           stationID,
		VendorID:            vendorID,
		CustomerID:          req.CustomerID,
		Coach:               coach,
		Berth:               berth,
		PassengerName:       passengerName,
		DeliveryWindowStart: dw.DeliveryWindowStart,
		DeliveryWindowEnd:   dw.DeliveryWindowEnd,
		Notes:               notes,
		Items:               lines,
	})
	if err != nil {
		writeTrainOrderError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, o)
}

func (s *Server) resolveWhatsAppDelivery(ctx context.Context, trainID, stationID uuid.UUID) (models.DeliveryWindow, error) {
	today := time.Now()
	for day := 0; day <= 2; day++ {
		journeyDate := today.AddDate(0, 0, day)
		dw, err := s.trains.ComputeDeliveryWindow(ctx, trainID, stationID, journeyDate)
		if err != nil {
			return models.DeliveryWindow{}, err
		}
		if dw.Feasible {
			return dw, nil
		}
	}
	// Best-effort: use tomorrow's computed window even if cutoff passed (dev / late orders)
	return s.trains.ComputeDeliveryWindow(ctx, trainID, stationID, today.AddDate(0, 0, 1))
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
