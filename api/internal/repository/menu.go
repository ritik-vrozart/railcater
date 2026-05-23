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

var validPortions = map[string]bool{
	"quarter": true, "half": true, "full": true, "single": true,
}

var validFoodTypes = map[string]bool{"veg": true, "non_veg": true}

type MenuRepository struct {
	pool *pgxpool.Pool
}

func NewMenuRepository(pool *pgxpool.Pool) *MenuRepository {
	return &MenuRepository{pool: pool}
}

// --- Categories (master) ---

func (r *MenuRepository) ListCategories(ctx context.Context, vendorID uuid.UUID, activeOnly bool) ([]models.MenuCategory, error) {
	where := "vendor_id = $1"
	args := []any{vendorID}
	if activeOnly {
		where += " AND is_active = true"
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, vendor_id, name, description, food_type, sort_order, is_active, created_at, updated_at
		FROM menu_categories WHERE `+where+` ORDER BY sort_order, name
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCategories(rows)
}

func (r *MenuRepository) GetCategory(ctx context.Context, vendorID, id uuid.UUID) (models.MenuCategory, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, vendor_id, name, description, food_type, sort_order, is_active, created_at, updated_at
		FROM menu_categories WHERE vendor_id = $1 AND id = $2
	`, vendorID, id)
	c, err := scanCategory(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.MenuCategory{}, apperror.ErrNotFound
	}
	return c, err
}

type CreateCategoryInput struct {
	VendorID    uuid.UUID
	Name        string
	Description *string
	FoodType    string
	SortOrder   int
}

func (r *MenuRepository) CreateCategory(ctx context.Context, in CreateCategoryInput) (models.MenuCategory, error) {
	ft := in.FoodType
	if ft == "" {
		ft = "veg"
	}
	if !validFoodTypes[ft] {
		return models.MenuCategory{}, apperror.BadRequest("food_type must be veg or non_veg")
	}
	row := r.pool.QueryRow(ctx, `
		INSERT INTO menu_categories (vendor_id, name, description, food_type, sort_order)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, vendor_id, name, description, food_type, sort_order, is_active, created_at, updated_at
	`, in.VendorID, in.Name, in.Description, ft, in.SortOrder)
	c, err := scanCategory(row)
	if err != nil {
		if isUniqueViolation(err) {
			return models.MenuCategory{}, apperror.ErrConflict
		}
		return models.MenuCategory{}, err
	}
	return c, nil
}

type UpdateCategoryInput struct {
	Name        *string
	Description *string
	FoodType    *string
	SortOrder   *int
	IsActive    *bool
}

func (r *MenuRepository) UpdateCategory(ctx context.Context, vendorID, id uuid.UUID, in UpdateCategoryInput) (models.MenuCategory, error) {
	if in.FoodType != nil && !validFoodTypes[*in.FoodType] {
		return models.MenuCategory{}, apperror.BadRequest("food_type must be veg or non_veg")
	}
	_, err := r.GetCategory(ctx, vendorID, id)
	if err != nil {
		return models.MenuCategory{}, err
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE menu_categories SET
			name = COALESCE($3, name),
			description = COALESCE($4, description),
			food_type = COALESCE($5, food_type),
			sort_order = COALESCE($6, sort_order),
			is_active = COALESCE($7, is_active),
			updated_at = now()
		WHERE vendor_id = $1 AND id = $2
	`, vendorID, id, in.Name, in.Description, in.FoodType, in.SortOrder, in.IsActive)
	if err != nil {
		return models.MenuCategory{}, err
	}
	return r.GetCategory(ctx, vendorID, id)
}

// --- Menu items + portions ---

type PortionInput struct {
	Portion       string
	Label         string
	PriceCents    int64
	StockQuantity int
	IsActive      bool
	SortOrder     int
}

type CreateMenuItemInput struct {
	VendorID    uuid.UUID
	CategoryID  *uuid.UUID
	Name        string
	Description *string
	ImageURL    *string
	IsVeg       bool
	IsActive    bool
	Portions    []PortionInput
}

func (r *MenuRepository) ListByVendor(ctx context.Context, vendorID uuid.UUID, activeOnly bool) ([]models.MenuItem, error) {
	where := "m.vendor_id = $1"
	args := []any{vendorID}
	if activeOnly {
		where += " AND m.is_active = true"
	}
	rows, err := r.pool.Query(ctx, `
		SELECT m.id, m.vendor_id, m.category_id, c.name, c.food_type, m.product_id,
		       m.name, m.description, m.image_url, m.price_cents, m.is_veg, m.is_active,
		       m.created_at, m.updated_at
		FROM menu_items m
		LEFT JOIN menu_categories c ON c.id = m.category_id
		WHERE `+where+`
		ORDER BY COALESCE(c.sort_order, 0), m.name
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.MenuItem
	var ids []uuid.UUID
	for rows.Next() {
		item, err := scanMenuItemRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
		ids = append(ids, item.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	portionMap, err := r.listPortionsByItems(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Portions = portionMap[items[i].ID]
		applyItemAggregates(&items[i])
	}
	return items, nil
}

func (r *MenuRepository) GetByID(ctx context.Context, vendorID, id uuid.UUID) (models.MenuItem, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT m.id, m.vendor_id, m.category_id, c.name, c.food_type, m.product_id,
		       m.name, m.description, m.image_url, m.price_cents, m.is_veg, m.is_active,
		       m.created_at, m.updated_at
		FROM menu_items m
		LEFT JOIN menu_categories c ON c.id = m.category_id
		WHERE m.vendor_id = $1 AND m.id = $2
	`, vendorID, id)
	item, err := scanMenuItemRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.MenuItem{}, apperror.ErrNotFound
	}
	if err != nil {
		return models.MenuItem{}, err
	}
	portions, err := r.listPortions(ctx, id)
	if err != nil {
		return models.MenuItem{}, err
	}
	item.Portions = portions
	applyItemAggregates(&item)
	return item, nil
}

func (r *MenuRepository) CreateItem(ctx context.Context, in CreateMenuItemInput) (models.MenuItem, error) {
	if len(in.Portions) == 0 {
		return models.MenuItem{}, apperror.BadRequest("at least one portion (quarter/half/full) is required")
	}
	if err := validatePortions(in.Portions); err != nil {
		return models.MenuItem{}, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.MenuItem{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	minPrice := in.Portions[0].PriceCents
	for _, p := range in.Portions {
		if p.PriceCents < minPrice {
			minPrice = p.PriceCents
		}
	}

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO menu_items (vendor_id, category_id, name, description, image_url, price_cents, is_veg, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, in.VendorID, in.CategoryID, in.Name, in.Description, in.ImageURL, minPrice, in.IsVeg, in.IsActive).Scan(&id)
	if err != nil {
		return models.MenuItem{}, err
	}

	if err := r.insertPortions(ctx, tx, id, in.Portions); err != nil {
		return models.MenuItem{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.MenuItem{}, err
	}
	return r.GetByID(ctx, in.VendorID, id)
}

type UpdateMenuItemInput struct {
	CategoryID  *uuid.UUID
	Name        *string
	Description *string
	ImageURL    *string
	IsVeg       *bool
	IsActive    *bool
	Portions    []PortionInput
}

func (r *MenuRepository) UpdateItem(ctx context.Context, vendorID, id uuid.UUID, in UpdateMenuItemInput) (models.MenuItem, error) {
	item, err := r.GetByID(ctx, vendorID, id)
	if err != nil {
		return models.MenuItem{}, err
	}
	if len(in.Portions) > 0 {
		if err := validatePortions(in.Portions); err != nil {
			return models.MenuItem{}, err
		}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.MenuItem{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	minPrice := item.PriceCents
	if len(in.Portions) > 0 {
		minPrice = in.Portions[0].PriceCents
		for _, p := range in.Portions {
			if p.PriceCents < minPrice {
				minPrice = p.PriceCents
			}
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE menu_items SET
			category_id = COALESCE($3, category_id),
			name = COALESCE($4, name),
			description = COALESCE($5, description),
			image_url = COALESCE($6, image_url),
			price_cents = $7,
			is_veg = COALESCE($8, is_veg),
			is_active = COALESCE($9, is_active),
			updated_at = now()
		WHERE vendor_id = $1 AND id = $2
	`, vendorID, id, in.CategoryID, in.Name, in.Description, in.ImageURL, minPrice, in.IsVeg, in.IsActive)
	if err != nil {
		return models.MenuItem{}, err
	}

	if len(in.Portions) > 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM menu_item_portions WHERE menu_item_id = $1`, id); err != nil {
			return models.MenuItem{}, err
		}
		if err := r.insertPortions(ctx, tx, id, in.Portions); err != nil {
			return models.MenuItem{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return models.MenuItem{}, err
	}
	return r.GetByID(ctx, vendorID, id)
}

func (r *MenuRepository) GetPortion(ctx context.Context, vendorID, portionID uuid.UUID) (models.MenuItemPortion, uuid.UUID, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT p.id, p.menu_item_id, p.portion, p.label, p.price_cents, p.stock_quantity,
		       p.is_active, p.sort_order, p.created_at, p.updated_at, m.vendor_id
		FROM menu_item_portions p
		JOIN menu_items m ON m.id = p.menu_item_id
		WHERE p.id = $1 AND m.vendor_id = $2
	`, portionID, vendorID)
	var p models.MenuItemPortion
	var vid uuid.UUID
	err := row.Scan(
		&p.ID, &p.MenuItemID, &p.Portion, &p.Label, &p.PriceCents, &p.StockQuantity,
		&p.IsActive, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt, &vid,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.MenuItemPortion{}, uuid.Nil, apperror.ErrNotFound
	}
	if err != nil {
		return models.MenuItemPortion{}, uuid.Nil, err
	}
	_ = vid // validated via WHERE m.vendor_id = $2
	return p, p.MenuItemID, nil
}

func (r *MenuRepository) DeductPortionStock(ctx context.Context, tx pgx.Tx, portionID uuid.UUID, qty int) error {
	tag, err := tx.Exec(ctx, `
		UPDATE menu_item_portions
		SET stock_quantity = stock_quantity - $2, updated_at = now()
		WHERE id = $1 AND stock_quantity >= $2
	`, portionID, qty)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperror.ErrInsufficient
	}
	return nil
}

func (r *MenuRepository) insertPortions(ctx context.Context, tx pgx.Tx, itemID uuid.UUID, portions []PortionInput) error {
	for _, p := range portions {
		_, err := tx.Exec(ctx, `
			INSERT INTO menu_item_portions (menu_item_id, portion, label, price_cents, stock_quantity, is_active, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, itemID, p.Portion, p.Label, p.PriceCents, p.StockQuantity, p.IsActive, p.SortOrder)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *MenuRepository) listPortions(ctx context.Context, itemID uuid.UUID) ([]models.MenuItemPortion, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, menu_item_id, portion, label, price_cents, stock_quantity, is_active, sort_order, created_at, updated_at
		FROM menu_item_portions WHERE menu_item_id = $1 ORDER BY sort_order, portion
	`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPortions(rows)
}

func (r *MenuRepository) listPortionsByItems(ctx context.Context, itemIDs []uuid.UUID) (map[uuid.UUID][]models.MenuItemPortion, error) {
	out := make(map[uuid.UUID][]models.MenuItemPortion)
	if len(itemIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, menu_item_id, portion, label, price_cents, stock_quantity, is_active, sort_order, created_at, updated_at
		FROM menu_item_portions WHERE menu_item_id = ANY($1) ORDER BY sort_order, portion
	`, itemIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		p, err := scanPortion(rows)
		if err != nil {
			return nil, err
		}
		out[p.MenuItemID] = append(out[p.MenuItemID], p)
	}
	return out, rows.Err()
}

func validatePortions(portions []PortionInput) error {
	seen := make(map[string]bool)
	for _, p := range portions {
		p.Portion = strings.ToLower(strings.TrimSpace(p.Portion))
		if !validPortions[p.Portion] {
			return apperror.BadRequest("invalid portion: must be quarter, half, full, or single")
		}
		if seen[p.Portion] {
			return apperror.BadRequest("duplicate portion: " + p.Portion)
		}
		seen[p.Portion] = true
		if p.PriceCents < 0 {
			return apperror.BadRequest("price must be non-negative")
		}
		if p.StockQuantity < 0 {
			return apperror.BadRequest("stock must be non-negative")
		}
		if strings.TrimSpace(p.Label) == "" {
			return apperror.BadRequest("portion label is required")
		}
	}
	return nil
}

func applyItemAggregates(item *models.MenuItem) {
	item.TotalStock = 0
	item.PriceCents = 0
	for _, p := range item.Portions {
		item.TotalStock += p.StockQuantity
		if item.PriceCents == 0 || p.PriceCents < item.PriceCents {
			item.PriceCents = p.PriceCents
		}
	}
}

func scanCategory(row pgx.Row) (models.MenuCategory, error) {
	var c models.MenuCategory
	err := row.Scan(&c.ID, &c.VendorID, &c.Name, &c.Description, &c.FoodType, &c.SortOrder, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func scanCategories(rows pgx.Rows) ([]models.MenuCategory, error) {
	var items []models.MenuCategory
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func scanMenuItemRow(row pgx.Row) (models.MenuItem, error) {
	var m models.MenuItem
	err := row.Scan(
		&m.ID, &m.VendorID, &m.CategoryID, &m.Category, &m.FoodType, &m.ProductID,
		&m.Name, &m.Description, &m.ImageURL, &m.PriceCents, &m.IsVeg, &m.IsActive,
		&m.CreatedAt, &m.UpdatedAt,
	)
	return m, err
}

func scanPortion(row pgx.Row) (models.MenuItemPortion, error) {
	var p models.MenuItemPortion
	err := row.Scan(
		&p.ID, &p.MenuItemID, &p.Portion, &p.Label, &p.PriceCents, &p.StockQuantity,
		&p.IsActive, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

func scanPortions(rows pgx.Rows) ([]models.MenuItemPortion, error) {
	var list []models.MenuItemPortion
	for rows.Next() {
		p, err := scanPortion(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}
