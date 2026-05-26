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

func (r *VendorRepository) ListForTrain(ctx context.Context, tenantID, trainID uuid.UUID) ([]models.Vendor, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT v.id, v.tenant_id, v.name, v.code, v.phone, v.is_active, v.is_approved, v.created_at, v.updated_at
		FROM vendors v
		JOIN vendor_trains vt ON vt.vendor_id = v.id AND vt.train_id = $2 AND vt.is_active
		WHERE v.tenant_id = $1 AND v.is_active AND v.is_approved
		ORDER BY v.name
	`, tenantID, trainID)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(items) > 0 {
		return items, nil
	}

	// Fallback: vendors serving any station on this train's route
	rows2, err := r.pool.Query(ctx, `
		SELECT DISTINCT v.id, v.tenant_id, v.name, v.code, v.phone, v.is_active, v.is_approved, v.created_at, v.updated_at
		FROM vendors v
		JOIN vendor_stations vs ON vs.vendor_id = v.id AND vs.is_active
		JOIN train_route_stops trs ON trs.station_id = vs.station_id AND trs.train_id = $2
		WHERE v.tenant_id = $1 AND v.is_active AND v.is_approved
		ORDER BY v.name
	`, tenantID, trainID)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	for rows2.Next() {
		v, err := scanVendor(rows2)
		if err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows2.Err()
}

func (r *VendorRepository) FirstStationOnTrainRoute(ctx context.Context, vendorID, trainID uuid.UUID) (uuid.UUID, error) {
	var stationID uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT vs.station_id
		FROM vendor_stations vs
		JOIN train_route_stops trs ON trs.station_id = vs.station_id AND trs.train_id = $2
		WHERE vs.vendor_id = $1 AND vs.is_active
		ORDER BY trs.stop_order
		LIMIT 1
	`, vendorID, trainID).Scan(&stationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, apperror.ErrNotFound
	}
	return stationID, err
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

type CreateVendorInput struct {
	TenantID uuid.UUID
	Name     string
	Code     string
	Phone    *string
}

func (r *VendorRepository) Create(ctx context.Context, in CreateVendorInput) (models.Vendor, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO vendors (tenant_id, name, code, phone, is_active, is_approved)
		VALUES ($1, $2, $3, $4, true, true)
		RETURNING id, tenant_id, name, code, phone, is_active, is_approved, created_at, updated_at
	`, in.TenantID, in.Name, in.Code, in.Phone)
	v, err := scanVendor(row)
	if err != nil && isUniqueViolation(err) {
		return models.Vendor{}, apperror.ErrConflict
	}
	return v, err
}

func (r *VendorRepository) LinkTrain(ctx context.Context, vendorID, trainID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO vendor_trains (vendor_id, train_id, is_active)
		VALUES ($1, $2, true)
		ON CONFLICT (vendor_id, train_id) DO UPDATE SET is_active = true
	`, vendorID, trainID)
	return err
}

func (r *VendorRepository) LinkStationsFromTrain(ctx context.Context, vendorID, trainID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO vendor_stations (vendor_id, station_id, is_active)
		SELECT $1, trs.station_id, true
		FROM train_route_stops trs
		WHERE trs.train_id = $2
		ON CONFLICT (vendor_id, station_id) DO UPDATE SET is_active = true
	`, vendorID, trainID)
	return err
}

func (r *VendorRepository) ListTrains(ctx context.Context, vendorID uuid.UUID) ([]models.VendorTrain, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.number, t.name, vt.is_active
		FROM vendor_trains vt
		JOIN trains t ON t.id = vt.train_id
		WHERE vt.vendor_id = $1
		ORDER BY t.number
	`, vendorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.VendorTrain
	for rows.Next() {
		var vt models.VendorTrain
		if err := rows.Scan(&vt.TrainID, &vt.TrainNumber, &vt.TrainName, &vt.IsActive); err != nil {
			return nil, err
		}
		items = append(items, vt)
	}
	if items == nil {
		items = []models.VendorTrain{}
	}
	return items, rows.Err()
}

func (r *VendorRepository) ListDetails(ctx context.Context, tenantID uuid.UUID, period DateRange) ([]models.VendorDetail, error) {
	vendors, err := r.List(ctx, tenantID, false)
	if err != nil {
		return nil, err
	}

	var out []models.VendorDetail
	for _, v := range vendors {
		detail, err := r.GetDetail(ctx, tenantID, v.ID, period)
		if err != nil {
			return nil, err
		}
		out = append(out, detail)
	}
	return out, nil
}

func (r *VendorRepository) orderStats(
	ctx context.Context, tenantID, vendorID uuid.UUID, period DateRange,
) (periodOrders int, periodRevenue int64, totalOrders int, totalRevenue int64, err error) {
	end := period.EndExclusive()
	err = r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE created_at >= $3 AND created_at < $4)::int,
			COALESCE(SUM(total_cents) FILTER (
				WHERE created_at >= $3 AND created_at < $4 AND status NOT IN ('cancelled')
			), 0)::bigint,
			COUNT(*)::int,
			COALESCE(SUM(total_cents) FILTER (WHERE status NOT IN ('cancelled')), 0)::bigint
		FROM orders
		WHERE vendor_id = $1 AND tenant_id = $2
	`, vendorID, tenantID, period.From, end).Scan(&periodOrders, &periodRevenue, &totalOrders, &totalRevenue)
	return
}

func (r *VendorRepository) GetDetail(ctx context.Context, tenantID, vendorID uuid.UUID, period DateRange) (models.VendorDetail, error) {
	v, err := r.GetByID(ctx, tenantID, vendorID)
	if err != nil {
		return models.VendorDetail{}, err
	}

	trains, err := r.ListTrains(ctx, vendorID)
	if err != nil {
		return models.VendorDetail{}, err
	}
	if trains == nil {
		trains = []models.VendorTrain{}
	}

	periodOrders, periodRevenue, totalOrders, totalRevenue, err := r.orderStats(ctx, tenantID, vendorID, period)
	if err != nil {
		return models.VendorDetail{}, err
	}

	var adminEmail *string
	_ = r.pool.QueryRow(ctx, `
		SELECT email FROM users
		WHERE vendor_id = $1 AND role = 'vendor_admin' AND is_active
		ORDER BY created_at LIMIT 1
	`, vendorID).Scan(&adminEmail)

	return models.VendorDetail{
		Vendor:        v,
		Trains:        trains,
		PeriodOrders:  periodOrders,
		PeriodRevenue: periodRevenue,
		TotalOrders:   totalOrders,
		TotalRevenue:  totalRevenue,
		AdminEmail:    adminEmail,
	}, nil
}

func (r *VendorRepository) AdminDashboard(ctx context.Context, tenantID uuid.UUID, period DateRange) (models.AdminDashboard, error) {
	var dash models.AdminDashboard
	err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE is_active AND is_approved)::int
		FROM vendors WHERE tenant_id = $1
	`, tenantID).Scan(&dash.TotalPantries, &dash.ActivePantries)
	if err != nil {
		return dash, err
	}

	end := period.EndExclusive()
	err = r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE created_at >= $2 AND created_at < $3)::int,
			COALESCE(SUM(total_cents) FILTER (
				WHERE created_at >= $2 AND created_at < $3 AND status NOT IN ('cancelled')
			), 0)::bigint,
			COUNT(*)::int,
			COALESCE(SUM(total_cents) FILTER (WHERE status NOT IN ('cancelled')), 0)::bigint
		FROM orders WHERE tenant_id = $1 AND vendor_id IS NOT NULL
	`, tenantID, period.From, end).Scan(&dash.PeriodOrders, &dash.PeriodRevenue, &dash.TotalOrders, &dash.TotalRevenue)
	if err != nil {
		return dash, err
	}

	dash.DateFrom = period.FromISO()
	dash.DateTo = period.ToISO()

	pantries, err := r.ListDetails(ctx, tenantID, period)
	if err != nil {
		return dash, err
	}
	if pantries == nil {
		pantries = []models.VendorDetail{}
	}
	dash.Pantries = pantries
	return dash, nil
}

func scanVendor(row pgx.Row) (models.Vendor, error) {
	var v models.Vendor
	err := row.Scan(&v.ID, &v.TenantID, &v.Name, &v.Code, &v.Phone, &v.IsActive, &v.IsApproved, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}
