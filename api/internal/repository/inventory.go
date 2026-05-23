package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/models"
)

type InventoryRepository struct {
	pool *pgxpool.Pool
}

func NewInventoryRepository(pool *pgxpool.Pool) *InventoryRepository {
	return &InventoryRepository{pool: pool}
}

func (r *InventoryRepository) Adjust(ctx context.Context, tenantID, productID uuid.UUID, delta int, reason string) (models.Product, error) {
	if delta == 0 {
		return models.Product{}, apperror.BadRequest("delta cannot be zero")
	}
	if reason != "restock" && reason != "adjustment" {
		return models.Product{}, apperror.BadRequest("reason must be restock or adjustment")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Product{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var exists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM products WHERE tenant_id = $1 AND id = $2)
	`, tenantID, productID).Scan(&exists)
	if err != nil {
		return models.Product{}, err
	}
	if !exists {
		return models.Product{}, apperror.ErrNotFound
	}

	var qty int
	err = tx.QueryRow(ctx, `SELECT quantity FROM inventory WHERE product_id = $1 FOR UPDATE`, productID).Scan(&qty)
	if err != nil {
		return models.Product{}, err
	}

	newQty := qty + delta
	if newQty < 0 {
		return models.Product{}, apperror.ErrInsufficient
	}

	_, err = tx.Exec(ctx, `
		UPDATE inventory SET quantity = $2, updated_at = now() WHERE product_id = $1
	`, productID, newQty)
	if err != nil {
		return models.Product{}, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO stock_movements (product_id, delta, reason) VALUES ($1, $2, $3)
	`, productID, delta, reason)
	if err != nil {
		return models.Product{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Product{}, err
	}

	prodRepo := NewProductRepository(r.pool)
	return prodRepo.GetByID(ctx, tenantID, productID)
}

func (r *InventoryRepository) ListMovements(ctx context.Context, tenantID, productID uuid.UUID, limit int) ([]models.StockMovement, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rows, err := r.pool.Query(ctx, `
		SELECT sm.id, sm.product_id, sm.delta, sm.reason, sm.reference_id, sm.created_at
		FROM stock_movements sm
		JOIN products p ON p.id = sm.product_id
		WHERE p.tenant_id = $1 AND sm.product_id = $2
		ORDER BY sm.created_at DESC
		LIMIT $3
	`, tenantID, productID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.StockMovement
	for rows.Next() {
		var m models.StockMovement
		if err := rows.Scan(&m.ID, &m.ProductID, &m.Delta, &m.Reason, &m.ReferenceID, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *InventoryRepository) ReserveStock(ctx context.Context, tx pgx.Tx, productID uuid.UUID, qty int) error {
	var available int
	err := tx.QueryRow(ctx, `
		SELECT quantity - reserved_quantity FROM inventory WHERE product_id = $1 FOR UPDATE
	`, productID).Scan(&available)
	if err != nil {
		return err
	}
	if available < qty {
		return apperror.ErrInsufficient
	}

	_, err = tx.Exec(ctx, `
		UPDATE inventory SET reserved_quantity = reserved_quantity + $2, updated_at = now()
		WHERE product_id = $1
	`, productID, qty)
	return err
}

func (r *InventoryRepository) CommitSale(ctx context.Context, tx pgx.Tx, productID uuid.UUID, qty int, orderID uuid.UUID) error {
	var qtyOnHand, reserved int
	err := tx.QueryRow(ctx, `
		SELECT quantity, reserved_quantity FROM inventory WHERE product_id = $1 FOR UPDATE
	`, productID).Scan(&qtyOnHand, &reserved)
	if err != nil {
		return err
	}
	available := qtyOnHand - reserved
	if available < qty {
		return apperror.ErrInsufficient
	}

	_, err = tx.Exec(ctx, `
		UPDATE inventory SET quantity = quantity - $2, updated_at = now()
		WHERE product_id = $1
	`, productID, qty)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO stock_movements (product_id, delta, reason, reference_id)
		VALUES ($1, $2, 'sale', $3)
	`, productID, -qty, orderID)
	return err
}
