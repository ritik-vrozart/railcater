package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/models"
)

type deliveryPayload struct {
	Secret              string  `json:"secret"`
	OrderID             string  `json:"order_id"`
	Phone               string  `json:"phone"`
	PassengerName       *string `json:"passenger_name,omitempty"`
	StationName         *string `json:"station_name,omitempty"`
	StationCode         *string `json:"station_code,omitempty"`
	Coach               *string `json:"coach,omitempty"`
	Berth               *string `json:"berth,omitempty"`
	DeliveryWindowStart string  `json:"delivery_window_start"`
	DeliveryWindowEnd   string  `json:"delivery_window_end"`
}

// DeliverySchedule asks the Python agent to send a WhatsApp message to the customer.
func DeliverySchedule(ctx context.Context, agentURL, secret string, order models.Order) {
	if agentURL == "" || secret == "" {
		return
	}
	if order.CustomerPhone == nil || strings.TrimSpace(*order.CustomerPhone) == "" {
		slog.Warn("delivery notify skipped: no customer phone", "order_id", order.ID)
		return
	}
	if order.DeliveryWindowStart == nil {
		slog.Warn("delivery notify skipped: no delivery window", "order_id", order.ID)
		return
	}

	end := ""
	if order.DeliveryWindowEnd != nil {
		end = order.DeliveryWindowEnd.Format(time.RFC3339)
	}

	body, err := json.Marshal(deliveryPayload{
		Secret:              secret,
		OrderID:             order.ID.String(),
		Phone:               strings.TrimSpace(*order.CustomerPhone),
		PassengerName:       order.PassengerName,
		StationName:         order.StationName,
		StationCode:         order.StationCode,
		Coach:               order.Coach,
		Berth:               order.Berth,
		DeliveryWindowStart: order.DeliveryWindowStart.Format(time.RFC3339),
		DeliveryWindowEnd:   end,
	})
	if err != nil {
		slog.Error("delivery notify marshal", "err", err)
		return
	}

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, agentURL, bytes.NewReader(body))
	if err != nil {
		slog.Error("delivery notify request", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("delivery notify post failed", "err", err, "url", agentURL)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		slog.Error("delivery notify bad status", "status", resp.StatusCode, "order_id", order.ID)
		return
	}
	slog.Info("delivery notify sent", "order_id", order.ID, "phone", *order.CustomerPhone)
}
