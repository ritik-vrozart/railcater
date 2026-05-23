package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/models"
)

type ProductRepository struct {
	pool *pgxpool.Pool
}

func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{pool: pool}
}

type CreateProductInput struct {
	TenantID    uuid.UUID
	SKU         string
	Name        string
	Description *string
	Unit        string
	PriceCents  int64
	Quantity    int
}

type UpdateProductInput struct {
	Name        *string
	Description *string
	Unit        *string
	PriceCents  *int64
	IsActive    *bool
}

func (r *ProductRepository) List(ctx context.Context, tenantID uuid.UUID, page, perPage int, activeOnly bool) ([]models.Product, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	where := "p.tenant_id = $1"
	args := []any{tenantID}
	if activeOnly {
		where += " AND p.is_active = true"
	}

	var total int
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM products p WHERE %s`, where)
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := fmt.Sprintf(`
		SELECT p.id, p.tenant_id, p.sku, p.name, p.description, p.unit, p.price_cents, p.is_active,
		       COALESCE(i.quantity, 0), COALESCE(i.reserved_quantity, 0), p.created_at, p.updated_at
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
		WHERE %s
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3
	`, where)
	args = append(args, perPage, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []models.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, p)
	}
	return items, total, rows.Err()
}

func (r *ProductRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (models.Product, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT p.id, p.tenant_id, p.sku, p.name, p.description, p.unit, p.price_cents, p.is_active,
		       COALESCE(i.quantity, 0), COALESCE(i.reserved_quantity, 0), p.created_at, p.updated_at
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
		WHERE p.tenant_id = $1 AND p.id = $2
	`, tenantID, id)

	p, err := scanProduct(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Product{}, apperror.ErrNotFound
	}
	return p, err
}

func (r *ProductRepository) Create(ctx context.Context, in CreateProductInput) (models.Product, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Product{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var p models.Product
	err = tx.QueryRow(ctx, `
		INSERT INTO products (tenant_id, sku, name, description, unit, price_cents)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, tenant_id, sku, name, description, unit, price_cents, is_active, created_at, updated_at
	`, in.TenantID, in.SKU, in.Name, in.Description, in.Unit, in.PriceCents).Scan(
		&p.ID, &p.TenantID, &p.SKU, &p.Name, &p.Description, &p.Unit, &p.PriceCents, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return models.Product{}, apperror.ErrConflict
		}
		return models.Product{}, err
	}

	qty := in.Quantity
	if qty < 0 {
		qty = 0
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO inventory (product_id, quantity) VALUES ($1, $2)
	`, p.ID, qty)
	if err != nil {
		return models.Product{}, err
	}
	if qty > 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO stock_movements (product_id, delta, reason) VALUES ($1, $2, 'restock')
		`, p.ID, qty)
		if err != nil {
			return models.Product{}, err
		}
	}

	p.Quantity = qty
	p.Reserved = 0

	if err := tx.Commit(ctx); err != nil {
		return models.Product{}, err
	}
	return p, nil
}

func (r *ProductRepository) Update(ctx context.Context, tenantID, id uuid.UUID, in UpdateProductInput) (models.Product, error) {
	_, err := r.GetByID(ctx, tenantID, id)
	if errors.Is(err, apperror.ErrNotFound) {
		return models.Product{}, err
	} else if err != nil {
		return models.Product{}, err
	}

	_, err = r.pool.Exec(ctx, `
		UPDATE products SET
			name = COALESCE($3, name),
			description = COALESCE($4, description),
			unit = COALESCE($5, unit),
			price_cents = COALESCE($6, price_cents),
			is_active = COALESCE($7, is_active),
			updated_at = now()
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id, in.Name, in.Description, in.Unit, in.PriceCents, in.IsActive)
	if err != nil {
		return models.Product{}, err
	}
	return r.GetByID(ctx, tenantID, id)
}

func scanProduct(row pgx.Row) (models.Product, error) {
	var p models.Product
	err := row.Scan(
		&p.ID, &p.TenantID, &p.SKU, &p.Name, &p.Description, &p.Unit, &p.PriceCents, &p.IsActive,
		&p.Quantity, &p.Reserved, &p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
