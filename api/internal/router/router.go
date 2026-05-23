package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/config"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/handler"
	appmw "github.com/ritikkumarpathak/whatsapp-bot/api/internal/middleware"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/repository"
)

func New(cfg config.Config, pool *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(appmw.Recoverer)
	r.Use(appmw.RequestLogger)
	r.Use(chimw.Timeout(60 * time.Second))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	health := handler.NewHealthHandler(pool)
	r.Get("/health", health.Health)
	r.Get("/ready", health.Ready)

	products := repository.NewProductRepository(pool)
	customers := repository.NewCustomerRepository(pool)
	inventory := repository.NewInventoryRepository(pool)
	orders := repository.NewOrderRepository(pool)
	stations := repository.NewStationRepository(pool)
	trains := repository.NewTrainRepository(pool)
	vendors := repository.NewVendorRepository(pool)
	menu := repository.NewMenuRepository(pool)
	pnr := repository.NewPNRRepository(pool)
	users := repository.NewUserRepository(pool)

	api := handler.NewServer(
		cfg.TenantID, cfg.JWTSecret, cfg.JWTExpiry,
		cfg.AgentNotifyURL, cfg.AgentNotifySecret,
		products, customers, inventory, orders, stations, trains, vendors, menu, pnr, users,
	)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", api.Register)
		r.Post("/auth/login", api.Login)
		r.With(appmw.Auth(cfg.JWTSecret)).Get("/auth/me", api.Me)

		// PNR & trains
		r.Get("/pnr/{pnr}", api.LookupPNR)
		r.Get("/stations", api.ListStations)
		r.Get("/trains", api.ListTrains)
		r.Get("/trains/{id}", api.GetTrain)
		r.Get("/trains/{id}/stations", api.GetTrainStations)
		r.Patch("/trains/{id}/delay", api.UpdateTrainDelay)

		// Vendors & menu
		r.Get("/vendors", api.ListVendors)
		r.Get("/vendors/{id}", api.GetVendor)
		r.Get("/stations/{stationId}/vendors", api.ListStationVendors)
		r.Get("/vendors/{vendorId}/menu", api.ListVendorMenu)
		r.Post("/vendors/{vendorId}/menu", api.CreateMenuItem)
		r.Put("/vendors/{vendorId}/menu/{itemId}", api.UpdateMenuItem)
		r.Get("/vendors/{vendorId}/menu/categories", api.ListMenuCategories)
		r.Post("/vendors/{vendorId}/menu/categories", api.CreateMenuCategory)
		r.Put("/vendors/{vendorId}/menu/categories/{categoryId}", api.UpdateMenuCategory)

		// Products & inventory (legacy / kitchen stock link)
		r.Get("/products", api.ListProducts)
		r.Post("/products", api.CreateProduct)
		r.Get("/products/{id}", api.GetProduct)
		r.Put("/products/{id}", api.UpdateProduct)
		r.Post("/products/{id}/stock", api.AdjustStock)
		r.Get("/products/{id}/stock/movements", api.ListStockMovements)

		r.Get("/customers", api.ListCustomers)
		r.Get("/customers/by-phone", api.GetCustomerByPhone)
		r.Post("/customers", api.CreateCustomer)
		r.Get("/customers/{id}", api.GetCustomer)
		r.Put("/customers/{id}", api.UpdateCustomer)

		// Orders
		r.Get("/orders", api.ListOrders)
		r.Post("/orders", api.CreateOrder)
		r.Post("/orders/train", api.CreateTrainOrder)
		r.Post("/orders/validate-delivery", api.ValidateDelivery)
		r.Post("/orders/whatsapp", api.CreateWhatsAppOrder)
		r.Get("/orders/{id}", api.GetOrder)
		r.Patch("/orders/{id}/status", api.UpdateOrderStatus)
		r.Patch("/orders/{id}/delivery", api.UpdateOrderDelivery)

		r.Post("/inventory/check", api.CheckStock)
	})

	return r
}
