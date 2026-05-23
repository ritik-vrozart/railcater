package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/apperror"
	"github.com/ritikkumarpathak/whatsapp-bot/api/internal/models"
)

type StationRepository struct {
	pool *pgxpool.Pool
}

func NewStationRepository(pool *pgxpool.Pool) *StationRepository {
	return &StationRepository{pool: pool}
}

func (r *StationRepository) List(ctx context.Context) ([]models.Station, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, code, name, city, state, created_at
		FROM stations ORDER BY code
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.Station
	for rows.Next() {
		s, err := scanStation(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

func (r *StationRepository) GetByID(ctx context.Context, id uuid.UUID) (models.Station, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, code, name, city, state, created_at FROM stations WHERE id = $1
	`, id)
	s, err := scanStation(row)
	if err == pgx.ErrNoRows {
		return models.Station{}, apperror.ErrNotFound
	}
	return s, err
}

func (r *StationRepository) GetByCode(ctx context.Context, code string) (models.Station, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, code, name, city, state, created_at FROM stations WHERE code = $1
	`, code)
	s, err := scanStation(row)
	if err == pgx.ErrNoRows {
		return models.Station{}, apperror.ErrNotFound
	}
	return s, err
}

func scanStation(row pgx.Row) (models.Station, error) {
	var s models.Station
	err := row.Scan(&s.ID, &s.Code, &s.Name, &s.City, &s.State, &s.CreatedAt)
	return s, err
}
