package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/models"
)

type DailyMenuRepository struct {
	pool *pgxpool.Pool
}

func NewDailyMenuRepository(pool *pgxpool.Pool) *DailyMenuRepository {
	return &DailyMenuRepository{pool: pool}
}

type DailyMenuItemInput struct {
	MenuItemID    uuid.UUID
	IsAvailable   bool
	StockOverride *int
}

func (r *DailyMenuRepository) GetOrCreate(ctx context.Context, vendorID uuid.UUID, menuDate time.Time) (models.DailyMenu, error) {
	dateStr := menuDate.Format("2006-01-02")
	row := r.pool.QueryRow(ctx, `
		INSERT INTO daily_menus (vendor_id, menu_date)
		VALUES ($1, $2::date)
		ON CONFLICT (vendor_id, menu_date) DO UPDATE SET updated_at = now()
		RETURNING id, vendor_id, menu_date::text, notes, created_at, updated_at
	`, vendorID, dateStr)

	var m models.DailyMenu
	err := row.Scan(&m.ID, &m.VendorID, &m.MenuDate, &m.Notes, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return models.DailyMenu{}, err
	}
	m.Items, err = r.listItems(ctx, m.ID)
	return m, err
}

func (r *DailyMenuRepository) Get(ctx context.Context, vendorID uuid.UUID, menuDate time.Time) (models.DailyMenu, error) {
	dateStr := menuDate.Format("2006-01-02")
	row := r.pool.QueryRow(ctx, `
		SELECT id, vendor_id, menu_date::text, notes, created_at, updated_at
		FROM daily_menus
		WHERE vendor_id = $1 AND menu_date = $2::date
	`, vendorID, dateStr)

	var m models.DailyMenu
	err := row.Scan(&m.ID, &m.VendorID, &m.MenuDate, &m.Notes, &m.CreatedAt, &m.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.DailyMenu{}, apperror.ErrNotFound
	}
	if err != nil {
		return models.DailyMenu{}, err
	}
	m.Items, err = r.listItems(ctx, m.ID)
	return m, err
}

func (r *DailyMenuRepository) SetItems(ctx context.Context, dailyMenuID uuid.UUID, items []DailyMenuItemInput) (models.DailyMenu, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.DailyMenu{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM daily_menu_items WHERE daily_menu_id = $1`, dailyMenuID); err != nil {
		return models.DailyMenu{}, err
	}

	for _, item := range items {
		_, err := tx.Exec(ctx, `
			INSERT INTO daily_menu_items (daily_menu_id, menu_item_id, is_available, stock_override)
			VALUES ($1, $2, $3, $4)
		`, dailyMenuID, item.MenuItemID, item.IsAvailable, item.StockOverride)
		if err != nil {
			return models.DailyMenu{}, err
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE daily_menus SET updated_at = now() WHERE id = $1`, dailyMenuID); err != nil {
		return models.DailyMenu{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.DailyMenu{}, err
	}

	var vendorID uuid.UUID
	var menuDate string
	err = r.pool.QueryRow(ctx, `
		SELECT vendor_id, menu_date::text FROM daily_menus WHERE id = $1
	`, dailyMenuID).Scan(&vendorID, &menuDate)
	if err != nil {
		return models.DailyMenu{}, err
	}

	d, _ := time.Parse("2006-01-02", menuDate)
	return r.Get(ctx, vendorID, d)
}

func (r *DailyMenuRepository) listItems(ctx context.Context, dailyMenuID uuid.UUID) ([]models.DailyMenuItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT dmi.id, dmi.daily_menu_id, dmi.menu_item_id, mi.name,
		       dmi.is_available, dmi.stock_override, dmi.created_at
		FROM daily_menu_items dmi
		JOIN menu_items mi ON mi.id = dmi.menu_item_id
		WHERE dmi.daily_menu_id = $1
		ORDER BY mi.name
	`, dailyMenuID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.DailyMenuItem
	for rows.Next() {
		var it models.DailyMenuItem
		if err := rows.Scan(
			&it.ID, &it.DailyMenuID, &it.MenuItemID, &it.MenuItemName,
			&it.IsAvailable, &it.StockOverride, &it.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (r *DailyMenuRepository) AvailableMenuItemIDs(ctx context.Context, vendorID uuid.UUID, menuDate time.Time) (map[uuid.UUID]bool, error) {
	m, err := r.Get(ctx, vendorID, menuDate)
	if errors.Is(err, apperror.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]bool)
	for _, it := range m.Items {
		if it.IsAvailable {
			out[it.MenuItemID] = true
		}
	}
	return out, nil
}
