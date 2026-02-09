package postgres

import (
	"context"
	"database/sql"

	"github.com/nandanugg/tj-test/module/core/domain"
	"github.com/nandanugg/tj-test/module/core/internal/repository/database"
)

var _ database.LocationRepository = (*LocationRepo)(nil)

type LocationRepo struct {
	db *sql.DB
}

func NewLocationRepo(db *sql.DB) *LocationRepo {
	return &LocationRepo{db: db}
}

func (r *LocationRepo) Insert(ctx context.Context, loc *domain.VehicleLocation) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO vehicle_locations (vehicle_id, latitude, longitude, timestamp) VALUES ($1, $2, $3, $4)`,
		loc.VehicleID, loc.Location.Lat, loc.Location.Lon, loc.Location.Timestamp,
	)
	return err
}

func (r *LocationRepo) GetLatest(ctx context.Context, vehicleID string) (*domain.VehicleLocation, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT vehicle_id, latitude, longitude, timestamp FROM vehicle_locations WHERE vehicle_id = $1 ORDER BY timestamp DESC LIMIT 1`,
		vehicleID,
	)

	var vl domain.VehicleLocation
	if err := row.Scan(&vl.VehicleID, &vl.Location.Lat, &vl.Location.Lon, &vl.Location.Timestamp); err != nil {
		return nil, err
	}
	return &vl, nil
}

func (r *LocationRepo) GetHistory(ctx context.Context, query *domain.HistoryQuery) ([]domain.VehicleLocation, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT vehicle_id, latitude, longitude, timestamp FROM vehicle_locations WHERE vehicle_id = $1 AND timestamp >= $2 AND timestamp <= $3 ORDER BY timestamp ASC`,
		query.VehicleID, query.Start, query.End,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []domain.VehicleLocation
	for rows.Next() {
		var vl domain.VehicleLocation
		if err := rows.Scan(&vl.VehicleID, &vl.Location.Lat, &vl.Location.Lon, &vl.Location.Timestamp); err != nil {
			return nil, err
		}
		results = append(results, vl)
	}
	return results, rows.Err()
}

func (r *LocationRepo) GetAllVehicles(ctx context.Context) ([]domain.Vehicle, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT vehicle_id FROM vehicle_locations ORDER BY vehicle_id`,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []domain.Vehicle
	for rows.Next() {
		var v domain.Vehicle
		if err := rows.Scan(&v.VehicleID); err != nil {
			return nil, err
		}
		results = append(results, v)
	}
	return results, rows.Err()
}
