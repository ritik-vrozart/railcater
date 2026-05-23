package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/models"
)

type CustomerRepository struct {
	pool *pgxpool.Pool
}

func NewCustomerRepository(pool *pgxpool.Pool) *CustomerRepository {
	return &CustomerRepository{pool: pool}
}

type CreateCustomerInput struct {
	TenantID          uuid.UUID
	Name              string
	Phone             *string
	Email             *string
	PreferredLanguage string
	Address           *string
}

type UpdateCustomerInput struct {
	Name              *string
	Phone             *string
	Email             *string
	PreferredLanguage *string
	Address           *string
}

func (r *CustomerRepository) List(ctx context.Context, tenantID uuid.UUID, page, perPage int, search string) ([]models.Customer, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	where := "tenant_id = $1"
	args := []any{tenantID}
	if search != "" {
		where += " AND (name ILIKE $2 OR phone ILIKE $2)"
		args = append(args, "%"+search+"%")
	}

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM customers WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitArg := len(args) + 1
	offsetArg := len(args) + 2
	listArgs := append(args, perPage, offset)
	q := fmt.Sprintf(`
		SELECT id, tenant_id, name, phone, email, preferred_language, address, created_at, updated_at
		FROM customers WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, limitArg, offsetArg)

	rows, err := r.pool.Query(ctx, q, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []models.Customer
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, c)
	}
	return items, total, rows.Err()
}

func (r *CustomerRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (models.Customer, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, phone, email, preferred_language, address, created_at, updated_at
		FROM customers WHERE tenant_id = $1 AND id = $2
	`, tenantID, id)

	c, err := scanCustomer(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Customer{}, apperror.ErrNotFound
	}
	return c, err
}

func (r *CustomerRepository) GetByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (models.Customer, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, phone, email, preferred_language, address, created_at, updated_at
		FROM customers WHERE tenant_id = $1 AND phone = $2
	`, tenantID, phone)

	c, err := scanCustomer(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Customer{}, apperror.ErrNotFound
	}
	return c, err
}

func (r *CustomerRepository) Create(ctx context.Context, in CreateCustomerInput) (models.Customer, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO customers (tenant_id, name, phone, email, preferred_language, address)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, tenant_id, name, phone, email, preferred_language, address, created_at, updated_at
	`, in.TenantID, in.Name, in.Phone, in.Email, in.PreferredLanguage, in.Address)
	return scanCustomer(row)
}

func (r *CustomerRepository) Update(ctx context.Context, tenantID, id uuid.UUID, in UpdateCustomerInput) (models.Customer, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE customers SET
			name = COALESCE($3, name),
			phone = COALESCE($4, phone),
			email = COALESCE($5, email),
			preferred_language = COALESCE($6, preferred_language),
			address = COALESCE($7, address),
			updated_at = now()
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id, in.Name, in.Phone, in.Email, in.PreferredLanguage, in.Address)
	if err != nil {
		return models.Customer{}, err
	}
	if tag.RowsAffected() == 0 {
		return models.Customer{}, apperror.ErrNotFound
	}
	return r.GetByID(ctx, tenantID, id)
}

func scanCustomer(row pgx.Row) (models.Customer, error) {
	var c models.Customer
	err := row.Scan(
		&c.ID, &c.TenantID, &c.Name, &c.Phone, &c.Email, &c.PreferredLanguage, &c.Address, &c.CreatedAt, &c.UpdatedAt,
	)
	return c, err
}
