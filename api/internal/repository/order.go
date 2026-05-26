package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/models"
)

type OrderRepository struct {
	pool      *pgxpool.Pool
	inventory *InventoryRepository
	products  *ProductRepository
	menu      *MenuRepository
}

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{
		pool:      pool,
		inventory: NewInventoryRepository(pool),
		products:  NewProductRepository(pool),
		menu:      NewMenuRepository(pool),
	}
}

type OrderLineInput struct {
	ProductID uuid.UUID
	Quantity  int
}

type CreateOrderInput struct {
	TenantID   uuid.UUID
	CustomerID *uuid.UUID
	Source     string
	Notes      *string
	Items      []OrderLineInput
}

type OrderListFilter struct {
	Status     string
	VendorID   *uuid.UUID
	CustomerID *uuid.UUID
	From       *time.Time
	ToEnd      *time.Time // exclusive end (start of day after To)
	TrainOnly  bool
}

func (r *OrderRepository) List(ctx context.Context, tenantID uuid.UUID, page, perPage int, f OrderListFilter) ([]models.Order, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	where := "o.tenant_id = $1"
	args := []any{tenantID}
	if f.Status != "" {
		args = append(args, f.Status)
		where += fmt.Sprintf(" AND o.status = $%d", len(args))
	}
	if f.VendorID != nil {
		args = append(args, *f.VendorID)
		where += fmt.Sprintf(" AND o.vendor_id = $%d", len(args))
	}
	if f.CustomerID != nil {
		args = append(args, *f.CustomerID)
		where += fmt.Sprintf(" AND o.customer_id = $%d", len(args))
	}
	if f.From != nil && f.ToEnd != nil {
		args = append(args, *f.From, *f.ToEnd)
		where += fmt.Sprintf(" AND o.created_at >= $%d AND o.created_at < $%d", len(args)-1, len(args))
	}
	if f.TrainOnly {
		where += " AND o.vendor_id IS NOT NULL"
	}

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM orders o WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	listArgs := append(args, perPage, offset)
	q := fmt.Sprintf(`
		SELECT o.id, o.tenant_id, o.customer_id, c.name, c.phone, o.status, o.source,
		       o.subtotal_cents, o.total_cents, o.notes,
		       o.pnr, o.train_id, tr.number, tr.name,
		       o.station_id, st.code, st.name,
		       o.vendor_id, v.name,
		       o.coach, o.berth, o.passenger_name,
		       o.delivery_window_start, o.delivery_window_end, o.delivery_notified_at,
		       o.expected_delivery_at, COALESCE(o.payment_status, 'pending'), o.payment_method,
		       o.created_at, o.updated_at
		FROM orders o
		LEFT JOIN customers c ON c.id = o.customer_id
		LEFT JOIN trains tr ON tr.id = o.train_id
		LEFT JOIN stations st ON st.id = o.station_id
		LEFT JOIN vendors v ON v.id = o.vendor_id
		WHERE %s
		ORDER BY o.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, limitIdx, offsetIdx)

	rows, err := r.pool.Query(ctx, q, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		o, err := scanOrderSummary(rows)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, o)
	}
	return orders, total, rows.Err()
}

func (r *OrderRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (models.Order, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT o.id, o.tenant_id, o.customer_id, c.name, c.phone, o.status, o.source,
		       o.subtotal_cents, o.total_cents, o.notes,
		       o.pnr, o.train_id, tr.number, tr.name,
		       o.station_id, st.code, st.name,
		       o.vendor_id, v.name,
		       o.coach, o.berth, o.passenger_name,
		       o.delivery_window_start, o.delivery_window_end, o.delivery_notified_at,
		       o.expected_delivery_at, COALESCE(o.payment_status, 'pending'), o.payment_method,
		       o.created_at, o.updated_at
		FROM orders o
		LEFT JOIN customers c ON c.id = o.customer_id
		LEFT JOIN trains tr ON tr.id = o.train_id
		LEFT JOIN stations st ON st.id = o.station_id
		LEFT JOIN vendors v ON v.id = o.vendor_id
		WHERE o.tenant_id = $1 AND o.id = $2
	`, tenantID, id)

	o, err := scanOrderSummary(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Order{}, apperror.ErrNotFound
	}
	if err != nil {
		return models.Order{}, err
	}

	items, err := r.listItems(ctx, id)
	if err != nil {
		return models.Order{}, err
	}
	o.Items = items
	return o, nil
}

func (r *OrderRepository) Create(ctx context.Context, in CreateOrderInput) (models.Order, error) {
	if len(in.Items) == 0 {
		return models.Order{}, apperror.BadRequest("order must have at least one item")
	}
	if in.Source == "" {
		in.Source = "dashboard"
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
		INSERT INTO orders (tenant_id, customer_id, status, source, notes)
		VALUES ($1, $2, 'confirmed', $3, $4)
		RETURNING id
	`, in.TenantID, in.CustomerID, in.Source, in.Notes).Scan(&orderID)
	if err != nil {
		return models.Order{}, err
	}

	var subtotal int64
	for _, line := range in.Items {
		if line.Quantity <= 0 {
			return models.Order{}, apperror.BadRequest("quantity must be positive")
		}

		var sku, name string
		var price int64
		var active bool
		err = tx.QueryRow(ctx, `
			SELECT sku, name, price_cents, is_active FROM products
			WHERE tenant_id = $1 AND id = $2
		`, in.TenantID, line.ProductID).Scan(&sku, &name, &price, &active)
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Order{}, apperror.BadRequest("product not found")
		} else if err != nil {
			return models.Order{}, err
		}
		if !active {
			return models.Order{}, apperror.BadRequest("product is inactive: " + sku)
		}

		if err := r.inventory.CommitSale(ctx, tx, line.ProductID, line.Quantity, orderID); err != nil {
			if errors.Is(err, apperror.ErrInsufficient) {
				return models.Order{}, apperror.Unprocessable("insufficient stock for " + sku)
			}
			return models.Order{}, err
		}

		lineTotal := price * int64(line.Quantity)
		subtotal += lineTotal

		_, err = tx.Exec(ctx, `
			INSERT INTO order_items (order_id, product_id, quantity, unit_price_cents, line_total_cents)
			VALUES ($1, $2, $3, $4, $5)
		`, orderID, line.ProductID, line.Quantity, price, lineTotal)
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

func (r *OrderRepository) UpdateDeliverySchedule(
	ctx context.Context,
	tenantID, id uuid.UUID,
	start, end time.Time,
	markNotified bool,
) (models.Order, error) {
	var notifiedClause string
	if markNotified {
		notifiedClause = ", delivery_notified_at = now()"
	}
	q := fmt.Sprintf(`
		UPDATE orders SET
			delivery_window_start = $3,
			delivery_window_end = $4,
			expected_delivery_at = $3,
			updated_at = now()
			%s
		WHERE tenant_id = $1 AND id = $2
	`, notifiedClause)

	tag, err := r.pool.Exec(ctx, q, tenantID, id, start, end)
	if err != nil {
		return models.Order{}, err
	}
	if tag.RowsAffected() == 0 {
		return models.Order{}, apperror.ErrNotFound
	}
	return r.GetByID(ctx, tenantID, id)
}

func (r *OrderRepository) UpdatePayment(
	ctx context.Context,
	tenantID, id uuid.UUID,
	paymentStatus string,
	paymentMethod *string,
) (models.Order, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE orders SET
			payment_status = $3,
			payment_method = $4,
			updated_at = now()
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id, paymentStatus, paymentMethod)
	if err != nil {
		return models.Order{}, err
	}
	if tag.RowsAffected() == 0 {
		return models.Order{}, apperror.ErrNotFound
	}
	return r.GetByID(ctx, tenantID, id)
}

func (r *OrderRepository) UpdateSeat(
	ctx context.Context,
	tenantID, id uuid.UUID,
	coach, berth string,
) (models.Order, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE orders SET coach = $3, berth = $4, updated_at = now()
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id, coach, berth)
	if err != nil {
		return models.Order{}, err
	}
	if tag.RowsAffected() == 0 {
		return models.Order{}, apperror.ErrNotFound
	}
	return r.GetByID(ctx, tenantID, id)
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) (models.Order, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE orders SET status = $3, updated_at = now()
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id, status)
	if err != nil {
		return models.Order{}, err
	}
	if tag.RowsAffected() == 0 {
		return models.Order{}, apperror.ErrNotFound
	}
	return r.GetByID(ctx, tenantID, id)
}

func (r *OrderRepository) listItems(ctx context.Context, orderID uuid.UUID) ([]models.OrderItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT oi.id, oi.order_id, oi.product_id, oi.menu_item_id, oi.menu_portion_id,
		       COALESCE(p.name, m.name), COALESCE(p.sku, ''),
		       mp.label, mp.portion,
		       oi.quantity, oi.unit_price_cents, oi.line_total_cents
		FROM order_items oi
		LEFT JOIN products p ON p.id = oi.product_id
		LEFT JOIN menu_items m ON m.id = oi.menu_item_id
		LEFT JOIN menu_item_portions mp ON mp.id = oi.menu_portion_id
		WHERE oi.order_id = $1
		ORDER BY oi.id
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		if err := rows.Scan(
			&item.ID, &item.OrderID, &item.ProductID, &item.MenuItemID, &item.MenuPortionID,
			&item.ProductName, &item.SKU, &item.PortionLabel, &item.Portion,
			&item.Quantity, &item.UnitPriceCents, &item.LineTotalCents,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanOrderSummary(row pgx.Row) (models.Order, error) {
	var o models.Order
	err := row.Scan(
		&o.ID, &o.TenantID, &o.CustomerID, &o.CustomerName, &o.CustomerPhone, &o.Status, &o.Source,
		&o.SubtotalCents, &o.TotalCents, &o.Notes,
		&o.PNR, &o.TrainID, &o.TrainNumber, &o.TrainName,
		&o.StationID, &o.StationCode, &o.StationName,
		&o.VendorID, &o.VendorName,
		&o.Coach, &o.Berth, &o.PassengerName,
		&o.DeliveryWindowStart, &o.DeliveryWindowEnd, &o.DeliveryNotifiedAt,
		&o.ExpectedDeliveryAt, &o.PaymentStatus, &o.PaymentMethod,
		&o.CreatedAt, &o.UpdatedAt,
	)
	return o, err
}
