package repository

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/models"
)

type PNRRepository struct {
	pool     *pgxpool.Pool
	trains   *TrainRepository
	stations *StationRepository
}

func NewPNRRepository(pool *pgxpool.Pool) *PNRRepository {
	return &PNRRepository{
		pool:     pool,
		trains:   NewTrainRepository(pool),
		stations: NewStationRepository(pool),
	}
}

func (r *PNRRepository) Lookup(ctx context.Context, tenantID uuid.UUID, pnr string) (models.PNRLookup, error) {
	pnr = strings.TrimSpace(strings.ToUpper(pnr))
	if len(pnr) != 10 {
		return models.PNRLookup{}, apperror.BadRequest("PNR must be 10 characters")
	}

	row := r.pool.QueryRow(ctx, `
		SELECT p.pnr, p.passenger_name, p.coach, p.berth, p.journey_date, p.booking_status,
		       p.from_station_id, p.to_station_id,
		       t.id, t.tenant_id, t.number, t.name, t.is_active, t.created_at, t.updated_at,
		       p.train_id
		FROM pnr_records p
		JOIN trains t ON t.id = p.train_id
		WHERE p.pnr = $1 AND t.tenant_id = $2
	`, pnr, tenantID)

	var lookup models.PNRLookup
	var journeyDate time.Time
	var fromID, toID, trainID uuid.UUID
	var train models.Train

	err := row.Scan(
		&lookup.PNR, &lookup.PassengerName, &lookup.Coach, &lookup.Berth,
		&journeyDate, &lookup.BookingStatus,
		&fromID, &toID,
		&train.ID, &train.TenantID, &train.Number, &train.Name, &train.IsActive, &train.CreatedAt, &train.UpdatedAt,
		&trainID,
	)
	if err == pgx.ErrNoRows {
		return models.PNRLookup{}, apperror.NotFound("PNR not found")
	}
	if err != nil {
		return models.PNRLookup{}, err
	}

	lookup.JourneyDate = journeyDate.Format("2006-01-02")
	lookup.Train = train

	if lookup.BookingStatus != "CONFIRMED" {
		return models.PNRLookup{}, apperror.Unprocessable("booking is not confirmed")
	}

	fromStation, err := r.stations.GetByID(ctx, fromID)
	if err != nil {
		return models.PNRLookup{}, err
	}
	toStation, err := r.stations.GetByID(ctx, toID)
	if err != nil {
		return models.PNRLookup{}, err
	}
	lookup.FromStation = fromStation
	lookup.ToStation = toStation

	stops, err := r.trains.ListOrderableStops(ctx, trainID, fromID, toID)
	if err != nil {
		return models.PNRLookup{}, err
	}
	lookup.AvailableStops = stops

	return lookup, nil
}
