package models

import (
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Unit        string    `json:"unit"`
	PriceCents  int64     `json:"price_cents"`
	IsActive    bool      `json:"is_active"`
	Quantity    int       `json:"quantity"`
	Reserved    int       `json:"reserved_quantity"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Customer struct {
	ID                uuid.UUID `json:"id"`
	TenantID          uuid.UUID `json:"tenant_id"`
	Name              string    `json:"name"`
	Phone             *string   `json:"phone,omitempty"`
	Email             *string   `json:"email,omitempty"`
	PreferredLanguage string    `json:"preferred_language"`
	Address           *string   `json:"address,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type OrderItem struct {
	ID             uuid.UUID  `json:"id"`
	OrderID        uuid.UUID  `json:"order_id"`
	ProductID      *uuid.UUID `json:"product_id,omitempty"`
	MenuItemID     *uuid.UUID `json:"menu_item_id,omitempty"`
	MenuPortionID  *uuid.UUID `json:"menu_portion_id,omitempty"`
	ProductName    string     `json:"product_name,omitempty"`
	PortionLabel   *string    `json:"portion_label,omitempty"`
	Portion        *string    `json:"portion,omitempty"`
	SKU            string     `json:"sku,omitempty"`
	Quantity       int        `json:"quantity"`
	UnitPriceCents int64      `json:"unit_price_cents"`
	LineTotalCents int64      `json:"line_total_cents"`
}

type Order struct {
	ID                   uuid.UUID   `json:"id"`
	TenantID             uuid.UUID   `json:"tenant_id"`
	CustomerID           *uuid.UUID  `json:"customer_id,omitempty"`
	CustomerName         *string     `json:"customer_name,omitempty"`
	CustomerPhone        *string     `json:"customer_phone,omitempty"`
	Status               string      `json:"status"`
	Source               string      `json:"source"`
	SubtotalCents        int64       `json:"subtotal_cents"`
	TotalCents           int64       `json:"total_cents"`
	Notes                *string     `json:"notes,omitempty"`
	PNR                  *string     `json:"pnr,omitempty"`
	TrainID              *uuid.UUID  `json:"train_id,omitempty"`
	TrainNumber          *string     `json:"train_number,omitempty"`
	TrainName            *string     `json:"train_name,omitempty"`
	StationID            *uuid.UUID  `json:"station_id,omitempty"`
	StationCode          *string     `json:"station_code,omitempty"`
	StationName          *string     `json:"station_name,omitempty"`
	VendorID             *uuid.UUID  `json:"vendor_id,omitempty"`
	VendorName           *string     `json:"vendor_name,omitempty"`
	Coach                *string     `json:"coach,omitempty"`
	Berth                *string     `json:"berth,omitempty"`
	PassengerName        *string     `json:"passenger_name,omitempty"`
	DeliveryWindowStart  *time.Time  `json:"delivery_window_start,omitempty"`
	DeliveryWindowEnd    *time.Time  `json:"delivery_window_end,omitempty"`
	DeliveryNotifiedAt   *time.Time  `json:"delivery_notified_at,omitempty"`
	Items                []OrderItem `json:"items,omitempty"`
	CreatedAt            time.Time   `json:"created_at"`
	UpdatedAt            time.Time   `json:"updated_at"`
}

type Station struct {
	ID        uuid.UUID `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	City      string    `json:"city"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
}

type Train struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Number    string    `json:"number"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TrainRouteStop struct {
	ID                 uuid.UUID `json:"id"`
	TrainID            uuid.UUID `json:"train_id"`
	StationID          uuid.UUID `json:"station_id"`
	StationCode        string    `json:"station_code"`
	StationName        string    `json:"station_name"`
	StopOrder          int       `json:"stop_order"`
	ScheduledArrival   *string   `json:"scheduled_arrival,omitempty"`
	ScheduledDeparture *string   `json:"scheduled_departure,omitempty"`
	HaltMinutes        int       `json:"halt_minutes"`
	Platform           *string   `json:"platform,omitempty"`
}

type TrainRun struct {
	ID            uuid.UUID `json:"id"`
	TrainID       uuid.UUID `json:"train_id"`
	RunDate       string    `json:"run_date"`
	DelayMinutes  int       `json:"delay_minutes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type TrainDetail struct {
	Train
	Stops []TrainRouteStop `json:"stops"`
	Run   *TrainRun        `json:"run,omitempty"`
}

type Vendor struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	Name       string    `json:"name"`
	Code       string    `json:"code"`
	Phone      *string   `json:"phone,omitempty"`
	IsActive   bool      `json:"is_active"`
	IsApproved bool      `json:"is_approved"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type VendorTrain struct {
	TrainID     uuid.UUID `json:"train_id"`
	TrainNumber string    `json:"train_number"`
	TrainName   string    `json:"train_name"`
	IsActive    bool      `json:"is_active"`
}

type VendorDetail struct {
	Vendor
	Trains          []VendorTrain `json:"trains"`
	PeriodOrders    int           `json:"period_orders"`
	PeriodRevenue   int64         `json:"period_revenue_cents"`
	TotalOrders     int           `json:"total_orders"`              // all-time
	TotalRevenue    int64         `json:"total_revenue_cents"`       // all-time
	AdminEmail      *string       `json:"admin_email,omitempty"`
}

type DailyMenu struct {
	ID        uuid.UUID       `json:"id"`
	VendorID  uuid.UUID       `json:"vendor_id"`
	MenuDate  string          `json:"menu_date"`
	Notes     *string         `json:"notes,omitempty"`
	Items     []DailyMenuItem `json:"items,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type DailyMenuItem struct {
	ID             uuid.UUID `json:"id"`
	DailyMenuID    uuid.UUID `json:"daily_menu_id"`
	MenuItemID     uuid.UUID `json:"menu_item_id"`
	MenuItemName   string    `json:"menu_item_name,omitempty"`
	IsAvailable    bool      `json:"is_available"`
	StockOverride  *int      `json:"stock_override,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type AdminDashboard struct {
	DateFrom       string         `json:"date_from"`
	DateTo         string         `json:"date_to"`
	TotalPantries  int            `json:"total_pantries"`
	ActivePantries int            `json:"active_pantries"`
	PeriodOrders   int            `json:"period_orders"`
	PeriodRevenue  int64          `json:"period_revenue_cents"`
	TotalOrders    int            `json:"total_orders"`
	TotalRevenue   int64          `json:"total_revenue_cents"`
	Pantries       []VendorDetail `json:"pantries"`
}

type MenuCategory struct {
	ID          uuid.UUID `json:"id"`
	VendorID    uuid.UUID `json:"vendor_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	FoodType    string    `json:"food_type"`
	SortOrder   int       `json:"sort_order"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type MenuItemPortion struct {
	ID            uuid.UUID `json:"id"`
	MenuItemID    uuid.UUID `json:"menu_item_id"`
	Portion       string    `json:"portion"`
	Label         string    `json:"label"`
	PriceCents    int64     `json:"price_cents"`
	StockQuantity int       `json:"stock_quantity"`
	IsActive      bool      `json:"is_active"`
	SortOrder     int       `json:"sort_order"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type MenuItem struct {
	ID          uuid.UUID         `json:"id"`
	VendorID    uuid.UUID         `json:"vendor_id"`
	CategoryID  *uuid.UUID        `json:"category_id,omitempty"`
	Category    *string           `json:"category,omitempty"`
	FoodType    *string           `json:"food_type,omitempty"`
	ProductID   *uuid.UUID        `json:"product_id,omitempty"`
	Name        string            `json:"name"`
	Description *string           `json:"description,omitempty"`
	ImageURL    *string           `json:"image_url,omitempty"`
	PriceCents  int64             `json:"price_cents"`
	IsVeg       bool              `json:"is_veg"`
	IsActive    bool              `json:"is_active"`
	Portions    []MenuItemPortion `json:"portions,omitempty"`
	TotalStock  int               `json:"total_stock"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type PNRLookup struct {
	PNR            string         `json:"pnr"`
	PassengerName  string         `json:"passenger_name"`
	Coach          string         `json:"coach"`
	Berth          string         `json:"berth"`
	JourneyDate    string         `json:"journey_date"`
	BookingStatus  string         `json:"booking_status"`
	Train          Train          `json:"train"`
	FromStation    Station        `json:"from_station"`
	ToStation      Station        `json:"to_station"`
	AvailableStops []TrainRouteStop `json:"available_stops"`
}

type DeliveryWindow struct {
	StationID             uuid.UUID `json:"station_id"`
	StationCode           string    `json:"station_code"`
	StationName           string    `json:"station_name"`
	EstimatedArrival      time.Time `json:"estimated_arrival"`
	EstimatedDeparture    time.Time `json:"estimated_departure"`
	DeliveryWindowStart   time.Time `json:"delivery_window_start"`
	DeliveryWindowEnd     time.Time `json:"delivery_window_end"`
	Feasible              bool      `json:"feasible"`
	FeasibilityMessage    string    `json:"feasibility_message,omitempty"`
}

type StockMovement struct {
	ID          uuid.UUID  `json:"id"`
	ProductID   uuid.UUID  `json:"product_id"`
	Delta       int        `json:"delta"`
	Reason      string     `json:"reason"`
	ReferenceID *uuid.UUID `json:"reference_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type User struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	VendorID  *uuid.UUID `json:"vendor_id,omitempty"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Phone     *string    `json:"phone,omitempty"`
	Role      string     `json:"role"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type PaginatedMeta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}
