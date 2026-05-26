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

type orderStatusPayload struct {
	Secret        string  `json:"secret"`
	OrderID       string  `json:"order_id"`
	Phone         string  `json:"phone"`
	PassengerName *string `json:"passenger_name,omitempty"`
	Status        string  `json:"status"`
	TrainNumber   *string `json:"train_number,omitempty"`
	Coach         *string `json:"coach,omitempty"`
	Berth         *string `json:"berth,omitempty"`
}

// OrderStatus asks the Python agent to notify the customer about order status changes.
func OrderStatus(ctx context.Context, agentURL, secret string, order models.Order, status string) {
	if agentURL == "" || secret == "" {
		return
	}
	if order.CustomerPhone == nil || strings.TrimSpace(*order.CustomerPhone) == "" {
		return
	}
	notifyStatuses := map[string]bool{
		"preparing": true, "ready": true, "dispatched": true, "delivered": true,
	}
	if !notifyStatuses[status] {
		return
	}

	body, err := json.Marshal(orderStatusPayload{
		Secret:        secret,
		OrderID:       order.ID.String(),
		Phone:         strings.TrimSpace(*order.CustomerPhone),
		PassengerName: order.PassengerName,
		Status:        status,
		TrainNumber:   order.TrainNumber,
		Coach:         order.Coach,
		Berth:         order.Berth,
	})
	if err != nil {
		slog.Error("order status notify marshal", "err", err)
		return
	}

	statusURL := strings.Replace(agentURL, "/delivery", "/order-status", 1)
	if statusURL == agentURL {
		statusURL = strings.TrimRight(agentURL, "/") + "/order-status"
	}

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, statusURL, bytes.NewReader(body))
	if err != nil {
		slog.Error("order status notify request", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("order status notify post failed", "err", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		slog.Error("order status notify bad status", "status", resp.StatusCode, "order_id", order.ID)
	}
}
