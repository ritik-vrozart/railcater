package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/models"
)

type TrainOrderLineInput struct {
	MenuPortionID uuid.UUID
	Quantity      int
}

type CreateTrainOrderInput struct {
	TenantID            uuid.UUID
	PNR                 string
	TrainID             uuid.UUID
	StationID           *uuid.UUID
	VendorID            uuid.UUID
	CustomerID          *uuid.UUID
	Coach               string
	Berth               string
	PassengerName       string
	DeliveryWindowStart time.Time
	DeliveryWindowEnd   time.Time
	Notes               *string
	Items               []TrainOrderLineInput
}

func (r *OrderRepository) CreateTrain(ctx context.Context, in CreateTrainOrderInput) (models.Order, error) {
	if len(in.Items) == 0 {
		return models.Order{}, apperror.BadRequest("order must have at least one item")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Order{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if in.CustomerID != nil {
		var ok bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM customers WHERE tenant_id = $1 AND id = $2)`, in.TenantID, *in.CustomerID).Scan(&ok)
		if err != nil {
			return models.Order{}, err
		}
		if !ok {
			return models.Order{}, apperror.BadRequest("customer not found")
		}
	}

	var orderID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (
			tenant_id, customer_id, status, source, notes,
			pnr, train_id, station_id, vendor_id, coach, berth, passenger_name,
			delivery_window_start, delivery_window_end
		)
		VALUES ($1, $2, 'confirmed', 'train', $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`, in.TenantID, in.CustomerID, in.Notes, in.PNR, in.TrainID, in.StationID, in.VendorID,
		in.Coach, in.Berth, in.PassengerName, in.DeliveryWindowStart, in.DeliveryWindowEnd,
	).Scan(&orderID)
	if err != nil {
		return models.Order{}, err
	}

	var subtotal int64
	for _, line := range in.Items {
		if line.Quantity <= 0 {
			return models.Order{}, apperror.BadRequest("quantity must be positive")
		}

		portion, menuItemID, err := r.menu.GetPortion(ctx, in.VendorID, line.MenuPortionID)
		if err != nil {
			return models.Order{}, err
		}
		if !portion.IsActive {
			return models.Order{}, apperror.BadRequest("portion is not available: " + portion.Label)
		}

		var itemName string
		var itemActive bool
		var productID *uuid.UUID
		err = tx.QueryRow(ctx, `
			SELECT name, is_active, product_id FROM menu_items WHERE id = $1 AND vendor_id = $2
		`, menuItemID, in.VendorID).Scan(&itemName, &itemActive, &productID)
		if err != nil {
			return models.Order{}, err
		}
		if !itemActive {
			return models.Order{}, apperror.BadRequest("menu item is inactive: " + itemName)
		}

		if err := r.menu.DeductPortionStock(ctx, tx, portion.ID, line.Quantity); err != nil {
			if errors.Is(err, apperror.ErrInsufficient) {
				return models.Order{}, apperror.Unprocessable("insufficient stock for " + itemName + " (" + portion.Label + ")")
			}
			return models.Order{}, err
		}

		if productID != nil {
			_ = r.inventory.CommitSale(ctx, tx, *productID, line.Quantity, orderID)
		}

		lineTotal := portion.PriceCents * int64(line.Quantity)
		subtotal += lineTotal

		_, err = tx.Exec(ctx, `
			INSERT INTO order_items (order_id, product_id, menu_item_id, menu_portion_id, quantity, unit_price_cents, line_total_cents)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, orderID, productID, menuItemID, portion.ID, line.Quantity, portion.PriceCents, lineTotal)
		if err != nil {
			return models.Order{}, err
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE orders SET subtotal_cents = $2, total_cents = $2, updated_at = now() WHERE id = $1
	`, orderID, subtotal)
	if err != nil {
		return models.Order{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Order{}, err
	}

	return r.GetByID(ctx, in.TenantID, orderID)
}
