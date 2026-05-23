package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/models"
)

const minOrderLeadMinutes = 90
const minHaltForDeliveryMinutes = 5

type TrainRepository struct {
	pool *pgxpool.Pool
}

func NewTrainRepository(pool *pgxpool.Pool) *TrainRepository {
	return &TrainRepository{pool: pool}
}

func (r *TrainRepository) List(ctx context.Context, tenantID uuid.UUID, activeOnly bool) ([]models.Train, error) {
	where := "tenant_id = $1"
	args := []any{tenantID}
	if activeOnly {
		where += " AND is_active = true"
	}

	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, tenant_id, number, name, is_active, created_at, updated_at
		FROM trains WHERE %s ORDER BY number
	`, where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.Train
	for rows.Next() {
		t, err := scanTrain(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

func (r *TrainRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (models.TrainDetail, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, number, name, is_active, created_at, updated_at
		FROM trains WHERE tenant_id = $1 AND id = $2
	`, tenantID, id)

	train, err := scanTrain(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.TrainDetail{}, apperror.ErrNotFound
	}
	if err != nil {
		return models.TrainDetail{}, err
	}

	stops, err := r.listStops(ctx, id)
	if err != nil {
		return models.TrainDetail{}, err
	}

	return models.TrainDetail{Train: train, Stops: stops}, nil
}

func (r *TrainRepository) GetRun(ctx context.Context, trainID uuid.UUID, runDate time.Time) (models.TrainRun, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, train_id, run_date, delay_minutes, created_at, updated_at
		FROM train_runs WHERE train_id = $1 AND run_date = $2
	`, trainID, runDate.Format("2006-01-02"))

	var run models.TrainRun
	var date time.Time
	err := row.Scan(&run.ID, &run.TrainID, &date, &run.DelayMinutes, &run.CreatedAt, &run.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.TrainRun{}, apperror.ErrNotFound
	}
	if err != nil {
		return models.TrainRun{}, err
	}
	run.RunDate = date.Format("2006-01-02")
	return run, nil
}

func (r *TrainRepository) UpsertRunDelay(ctx context.Context, trainID uuid.UUID, runDate time.Time, delayMinutes int) (models.TrainRun, error) {
	if delayMinutes < 0 {
		return models.TrainRun{}, apperror.BadRequest("delay_minutes must be non-negative")
	}

	row := r.pool.QueryRow(ctx, `
		INSERT INTO train_runs (train_id, run_date, delay_minutes)
		VALUES ($1, $2, $3)
		ON CONFLICT (train_id, run_date) DO UPDATE SET
			delay_minutes = EXCLUDED.delay_minutes,
			updated_at = now()
		RETURNING id, train_id, run_date, delay_minutes, created_at, updated_at
	`, trainID, runDate.Format("2006-01-02"), delayMinutes)

	var run models.TrainRun
	var date time.Time
	if err := row.Scan(&run.ID, &run.TrainID, &date, &run.DelayMinutes, &run.CreatedAt, &run.UpdatedAt); err != nil {
		return models.TrainRun{}, err
	}
	run.RunDate = date.Format("2006-01-02")
	return run, nil
}

func (r *TrainRepository) ListOrderableStops(ctx context.Context, trainID, fromStationID, toStationID uuid.UUID) ([]models.TrainRouteStop, error) {
	rows, err := r.pool.Query(ctx, `
		WITH bounds AS (
			SELECT
				(SELECT stop_order FROM train_route_stops WHERE train_id = $1 AND station_id = $2) AS from_ord,
				(SELECT stop_order FROM train_route_stops WHERE train_id = $1 AND station_id = $3) AS to_ord
		)
		SELECT trs.id, trs.train_id, trs.station_id, s.code, s.name,
		       trs.stop_order,
		       trs.scheduled_arrival::text, trs.scheduled_departure::text,
		       trs.halt_minutes, trs.platform
		FROM train_route_stops trs
		JOIN stations s ON s.id = trs.station_id
		CROSS JOIN bounds b
		WHERE trs.train_id = $1
		  AND trs.stop_order > b.from_ord
		  AND trs.stop_order <= b.to_ord
		  AND trs.halt_minutes >= $4
		  AND EXISTS (
		      SELECT 1 FROM vendor_stations vs
		      JOIN vendors v ON v.id = vs.vendor_id AND v.is_active AND v.is_approved
		      WHERE vs.station_id = trs.station_id AND vs.is_active
		  )
		ORDER BY trs.stop_order
	`, trainID, fromStationID, toStationID, minHaltForDeliveryMinutes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRouteStops(rows)
}

func (r *TrainRepository) GetRouteStop(ctx context.Context, trainID, stationID uuid.UUID) (models.TrainRouteStop, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT trs.id, trs.train_id, trs.station_id, s.code, s.name,
		       trs.stop_order,
		       trs.scheduled_arrival::text, trs.scheduled_departure::text,
		       trs.halt_minutes, trs.platform
		FROM train_route_stops trs
		JOIN stations s ON s.id = trs.station_id
		WHERE trs.train_id = $1 AND trs.station_id = $2
	`, trainID, stationID)

	stop, err := scanRouteStop(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.TrainRouteStop{}, apperror.ErrNotFound
	}
	return stop, err
}

func (r *TrainRepository) IsStopOnJourney(ctx context.Context, trainID, stationID, fromStationID, toStationID uuid.UUID) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		WITH bounds AS (
			SELECT
				(SELECT stop_order FROM train_route_stops WHERE train_id = $1 AND station_id = $3) AS from_ord,
				(SELECT stop_order FROM train_route_stops WHERE train_id = $1 AND station_id = $4) AS to_ord
		)
		SELECT EXISTS (
			SELECT 1 FROM train_route_stops trs, bounds b
			WHERE trs.train_id = $1 AND trs.station_id = $2
			  AND trs.stop_order > b.from_ord AND trs.stop_order <= b.to_ord
		)
	`, trainID, stationID, fromStationID, toStationID).Scan(&ok)
	return ok, err
}

func (r *TrainRepository) ComputeDeliveryWindow(
	ctx context.Context,
	trainID, stationID uuid.UUID,
	journeyDate time.Time,
) (models.DeliveryWindow, error) {
	stop, err := r.GetRouteStop(ctx, trainID, stationID)
	if err != nil {
		return models.DeliveryWindow{}, err
	}

	delay := 0
	run, err := r.GetRun(ctx, trainID, journeyDate)
	if err == nil {
		delay = run.DelayMinutes
	} else if !errors.Is(err, apperror.ErrNotFound) {
		return models.DeliveryWindow{}, err
	}

	arrival, departure, err := stopDateTimes(journeyDate, stop, delay)
	if err != nil {
		return models.DeliveryWindow{}, apperror.BadRequest(err.Error())
	}

	now := time.Now()
	windowStart := arrival.Add(-30 * time.Minute)
	windowEnd := departure.Add(-10 * time.Minute)
	orderCutoff := arrival.Add(-time.Duration(minOrderLeadMinutes) * time.Minute)

	dw := models.DeliveryWindow{
		StationID:           stationID,
		StationCode:         stop.StationCode,
		StationName:         stop.StationName,
		EstimatedArrival:    arrival,
		EstimatedDeparture:  departure,
		DeliveryWindowStart: windowStart,
		DeliveryWindowEnd:   windowEnd,
	}

	switch {
	case stop.HaltMinutes < minHaltForDeliveryMinutes:
		dw.Feasible = false
		dw.FeasibilityMessage = "station halt too short for food delivery"
	case now.After(windowEnd):
		dw.Feasible = false
		dw.FeasibilityMessage = "delivery window has passed for this station"
	case now.After(orderCutoff):
		dw.Feasible = false
		dw.FeasibilityMessage = fmt.Sprintf("order cutoff passed (must order at least %d minutes before arrival)", minOrderLeadMinutes)
	default:
		dw.Feasible = true
	}

	return dw, nil
}

func (r *TrainRepository) listStops(ctx context.Context, trainID uuid.UUID) ([]models.TrainRouteStop, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT trs.id, trs.train_id, trs.station_id, s.code, s.name,
		       trs.stop_order,
		       trs.scheduled_arrival::text, trs.scheduled_departure::text,
		       trs.halt_minutes, trs.platform
		FROM train_route_stops trs
		JOIN stations s ON s.id = trs.station_id
		WHERE trs.train_id = $1
		ORDER BY trs.stop_order
	`, trainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRouteStops(rows)
}

func scanTrain(row pgx.Row) (models.Train, error) {
	var t models.Train
	err := row.Scan(&t.ID, &t.TenantID, &t.Number, &t.Name, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func scanRouteStop(row pgx.Row) (models.TrainRouteStop, error) {
	var s models.TrainRouteStop
	err := row.Scan(
		&s.ID, &s.TrainID, &s.StationID, &s.StationCode, &s.StationName,
		&s.StopOrder, &s.ScheduledArrival, &s.ScheduledDeparture,
		&s.HaltMinutes, &s.Platform,
	)
	return s, err
}

func scanRouteStops(rows pgx.Rows) ([]models.TrainRouteStop, error) {
	var stops []models.TrainRouteStop
	for rows.Next() {
		s, err := scanRouteStop(rows)
		if err != nil {
			return nil, err
		}
		stops = append(stops, s)
	}
	return stops, rows.Err()
}

func stopDateTimes(journeyDate time.Time, stop models.TrainRouteStop, delayMinutes int) (arrival, departure time.Time, err error) {
	loc := journeyDate.Location()

	if stop.ScheduledArrival != nil && *stop.ScheduledArrival != "" {
		arrival, err = combineDateAndTime(journeyDate, *stop.ScheduledArrival, loc)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	if stop.ScheduledDeparture != nil && *stop.ScheduledDeparture != "" {
		departure, err = combineDateAndTime(journeyDate, *stop.ScheduledDeparture, loc)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		if !arrival.IsZero() && !departure.IsZero() && !arrival.Before(departure) {
			departure = departure.Add(24 * time.Hour)
		}
	}

	if arrival.IsZero() && !departure.IsZero() {
		arrival = departure.Add(-time.Duration(stop.HaltMinutes) * time.Minute)
	}
	if departure.IsZero() && !arrival.IsZero() {
		departure = arrival.Add(time.Duration(stop.HaltMinutes) * time.Minute)
	}
	if arrival.IsZero() || departure.IsZero() {
		return time.Time{}, time.Time{}, fmt.Errorf("station %s has no schedule times", stop.StationCode)
	}

	delay := time.Duration(delayMinutes) * time.Minute
	return arrival.Add(delay), departure.Add(delay), nil
}

func combineDateAndTime(date time.Time, timeStr string, loc *time.Location) (time.Time, error) {
	parsed, err := time.ParseInLocation("15:04:05", timeStr, loc)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(date.Year(), date.Month(), date.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), 0, loc), nil
}
