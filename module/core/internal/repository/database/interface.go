package database

import (
	"context"

	"github.com/nandanugg/tj-test/module/core/domain"
)

type LocationRepository interface {
	Insert(ctx context.Context, loc *domain.VehicleLocation) error
	GetLatest(ctx context.Context, vehicleID string) (*domain.VehicleLocation, error)
	GetHistory(ctx context.Context, query *domain.HistoryQuery) ([]domain.VehicleLocation, error)
	GetAllVehicles(ctx context.Context) ([]domain.Vehicle, error)
}
