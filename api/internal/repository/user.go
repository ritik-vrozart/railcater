package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/models"
)

var validRoles = map[string]bool{
	"super_admin": true, "vendor_admin": true, "passenger": true,
}

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

type CreateUserInput struct {
	TenantID     uuid.UUID
	Name         string
	Email        string
	Phone        *string
	PasswordHash string
	Role         string
	VendorID     *uuid.UUID
}

func (r *UserRepository) Create(ctx context.Context, in CreateUserInput) (models.User, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	role := in.Role
	if role == "" {
		role = "passenger"
	}
	if !validRoles[role] {
		return models.User{}, apperror.BadRequest("invalid role")
	}

	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (tenant_id, name, email, phone, password_hash, role, vendor_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, tenant_id, vendor_id, name, email, phone, role, is_active, created_at, updated_at
	`, in.TenantID, strings.TrimSpace(in.Name), email, in.Phone, in.PasswordHash, role, in.VendorID)

	u, err := scanUser(row)
	if err != nil {
		if isUniqueViolation(err) {
			return models.User{}, apperror.ErrConflict
		}
		return models.User{}, err
	}
	return u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (models.User, string, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, vendor_id, name, email, phone, role, is_active, created_at, updated_at, password_hash
		FROM users WHERE tenant_id = $1 AND email = $2
	`, tenantID, strings.ToLower(strings.TrimSpace(email)))

	var hash string
	u, err := scanUserWithHash(row, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, "", apperror.ErrNotFound
	}
	return u, hash, err
}

func (r *UserRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (models.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, vendor_id, name, email, phone, role, is_active, created_at, updated_at
		FROM users WHERE tenant_id = $1 AND id = $2
	`, tenantID, id)

	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, apperror.ErrNotFound
	}
	return u, err
}

func scanUser(row pgx.Row) (models.User, error) {
	var u models.User
	err := row.Scan(
		&u.ID, &u.TenantID, &u.VendorID, &u.Name, &u.Email, &u.Phone,
		&u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	return u, err
}

func scanUserWithHash(row pgx.Row, hash *string) (models.User, error) {
	var u models.User
	err := row.Scan(
		&u.ID, &u.TenantID, &u.VendorID, &u.Name, &u.Email, &u.Phone,
		&u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, hash,
	)
	return u, err
}
