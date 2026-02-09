package service

import (
	"context"

	"github.com/nandanugg/tj-test/module/core/domain"
	"github.com/nandanugg/tj-test/module/core/internal/repository/database"
)

type LocationService struct {
	repo database.LocationRepository
}

func NewLocationService(repo database.LocationRepository) *LocationService {
	return &LocationService{repo: repo}
}

func (s *LocationService) SaveLocation(ctx context.Context, vl *domain.VehicleLocation) error {
	return s.repo.Insert(ctx, vl)
}

func (s *LocationService) GetLatest(ctx context.Context, vehicleID string) (*domain.VehicleLocation, error) {
	return s.repo.GetLatest(ctx, vehicleID)
}

func (s *LocationService) GetHistory(ctx context.Context, query *domain.HistoryQuery) ([]domain.VehicleLocation, error) {
	return s.repo.GetHistory(ctx, query)
}

func (s *LocationService) GetAllVehicles(ctx context.Context) ([]domain.Vehicle, error) {
	return s.repo.GetAllVehicles(ctx)
}
