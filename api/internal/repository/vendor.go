package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/models"
)

type VendorRepository struct {
	pool *pgxpool.Pool
}

func NewVendorRepository(pool *pgxpool.Pool) *VendorRepository {
	return &VendorRepository{pool: pool}
}

func (r *VendorRepository) List(ctx context.Context, tenantID uuid.UUID, approvedOnly bool) ([]models.Vendor, error) {
	where := "tenant_id = $1"
	args := []any{tenantID}
	if approvedOnly {
		where += " AND is_approved = true AND is_active = true"
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, name, code, phone, is_active, is_approved, created_at, updated_at
		FROM vendors WHERE `+where+` ORDER BY name
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.Vendor
	for rows.Next() {
		v, err := scanVendor(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}

func (r *VendorRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (models.Vendor, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, code, phone, is_active, is_approved, created_at, updated_at
		FROM vendors WHERE tenant_id = $1 AND id = $2
	`, tenantID, id)

	v, err := scanVendor(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Vendor{}, apperror.ErrNotFound
	}
	return v, err
}

func (r *VendorRepository) ListAtStation(ctx context.Context, tenantID, stationID uuid.UUID) ([]models.Vendor, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT v.id, v.tenant_id, v.name, v.code, v.phone, v.is_active, v.is_approved, v.created_at, v.updated_at
		FROM vendors v
		JOIN vendor_stations vs ON vs.vendor_id = v.id
		WHERE v.tenant_id = $1 AND vs.station_id = $2 AND vs.is_active
		  AND v.is_active AND v.is_approved
		ORDER BY v.name
	`, tenantID, stationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.Vendor
	for rows.Next() {
		v, err := scanVendor(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}

func (r *VendorRepository) ServesStation(ctx context.Context, vendorID, stationID uuid.UUID) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM vendor_stations vs
			JOIN vendors v ON v.id = vs.vendor_id
			WHERE vs.vendor_id = $1 AND vs.station_id = $2 AND vs.is_active
			  AND v.is_active AND v.is_approved
		)
	`, vendorID, stationID).Scan(&ok)
	return ok, err
}

func scanVendor(row pgx.Row) (models.Vendor, error) {
	var v models.Vendor
	err := row.Scan(&v.ID, &v.TenantID, &v.Name, &v.Code, &v.Phone, &v.IsActive, &v.IsApproved, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}
