package handler

import (
	"time"

	"github.com/google/uuid"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/repository"
)

type Server struct {
	tenantID          uuid.UUID
	jwtSecret         string
	jwtExpiry         time.Duration
	agentNotifyURL    string
	agentNotifySecret string
	products   *repository.ProductRepository
	customers  *repository.CustomerRepository
	inventory  *repository.InventoryRepository
	orders     *repository.OrderRepository
	stations   *repository.StationRepository
	trains     *repository.TrainRepository
	vendors    *repository.VendorRepository
	menu       *repository.MenuRepository
	pnr        *repository.PNRRepository
	users      *repository.UserRepository
}

func NewServer(
	tenantID uuid.UUID,
	jwtSecret string,
	jwtExpiry time.Duration,
	agentNotifyURL string,
	agentNotifySecret string,
	products *repository.ProductRepository,
	customers *repository.CustomerRepository,
	inventory *repository.InventoryRepository,
	orders *repository.OrderRepository,
	stations *repository.StationRepository,
	trains *repository.TrainRepository,
	vendors *repository.VendorRepository,
	menu *repository.MenuRepository,
	pnr *repository.PNRRepository,
	users *repository.UserRepository,
) *Server {
	return &Server{
		tenantID:          tenantID,
		jwtSecret:         jwtSecret,
		jwtExpiry:         jwtExpiry,
		agentNotifyURL:    agentNotifyURL,
		agentNotifySecret: agentNotifySecret,
		products:   products,
		customers:  customers,
		inventory:  inventory,
		orders:     orders,
		stations:   stations,
		trains:     trains,
		vendors:    vendors,
		menu:       menu,
		pnr:        pnr,
		users:      users,
	}
}
