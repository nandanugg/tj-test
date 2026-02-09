package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nandanugg/tj-test/module/core/domain"
)

type mockLocationRepo struct {
	insertFn         func(ctx context.Context, loc *domain.VehicleLocation) error
	getLatestFn      func(ctx context.Context, vehicleID string) (*domain.VehicleLocation, error)
	getHistoryFn     func(ctx context.Context, query *domain.HistoryQuery) ([]domain.VehicleLocation, error)
	getAllVehiclesFn func(ctx context.Context) ([]domain.Vehicle, error)
}

func (m *mockLocationRepo) Insert(ctx context.Context, loc *domain.VehicleLocation) error {
	return m.insertFn(ctx, loc)
}

func (m *mockLocationRepo) GetLatest(ctx context.Context, vehicleID string) (*domain.VehicleLocation, error) {
	return m.getLatestFn(ctx, vehicleID)
}

func (m *mockLocationRepo) GetHistory(ctx context.Context, query *domain.HistoryQuery) ([]domain.VehicleLocation, error) {
	return m.getHistoryFn(ctx, query)
}

func (m *mockLocationRepo) GetAllVehicles(ctx context.Context) ([]domain.Vehicle, error) {
	return m.getAllVehiclesFn(ctx)
}

func TestSaveLocation_Success(t *testing.T) {
	var inserted *domain.VehicleLocation
	repo := &mockLocationRepo{
		insertFn: func(_ context.Context, loc *domain.VehicleLocation) error {
			inserted = loc
			return nil
		},
	}

	svc := NewLocationService(repo)
	vl := &domain.VehicleLocation{
		VehicleID: "B1234XYZ",
		Location: domain.Location{
			Lat:       -6.2088,
			Lon:       106.8456,
			Timestamp: time.Unix(1715003456, 0),
		},
	}

	err := svc.SaveLocation(context.Background(), vl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inserted == nil {
		t.Fatal("expected Insert to be called")
	}
	if inserted.VehicleID != "B1234XYZ" {
		t.Errorf("expected B1234XYZ, got %s", inserted.VehicleID)
	}
}

func TestSaveLocation_RepoError(t *testing.T) {
	repo := &mockLocationRepo{
		insertFn: func(_ context.Context, _ *domain.VehicleLocation) error {
			return errors.New("db error")
		},
	}

	svc := NewLocationService(repo)
	err := svc.SaveLocation(context.Background(), &domain.VehicleLocation{VehicleID: "X"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetLatest_Success(t *testing.T) {
	ts := time.Unix(1715003456, 0)
	repo := &mockLocationRepo{
		getLatestFn: func(_ context.Context, vehicleID string) (*domain.VehicleLocation, error) {
			return &domain.VehicleLocation{
				VehicleID: vehicleID,
				Location:  domain.Location{Lat: -6.2088, Lon: 106.8456, Timestamp: ts},
			}, nil
		},
	}

	svc := NewLocationService(repo)
	result, err := svc.GetLatest(context.Background(), "B1234XYZ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.VehicleID != "B1234XYZ" {
		t.Errorf("expected B1234XYZ, got %s", result.VehicleID)
	}
	if result.Location.Lat != -6.2088 {
		t.Errorf("expected -6.2088, got %f", result.Location.Lat)
	}
}

func TestGetLatest_NotFound(t *testing.T) {
	repo := &mockLocationRepo{
		getLatestFn: func(_ context.Context, _ string) (*domain.VehicleLocation, error) {
			return nil, errors.New("not found")
		},
	}

	svc := NewLocationService(repo)
	_, err := svc.GetLatest(context.Background(), "UNKNOWN")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetHistory_Success(t *testing.T) {
	ts1 := time.Unix(1715000000, 0)
	ts2 := time.Unix(1715005000, 0)
	repo := &mockLocationRepo{
		getHistoryFn: func(_ context.Context, query *domain.HistoryQuery) ([]domain.VehicleLocation, error) {
			return []domain.VehicleLocation{
				{VehicleID: query.VehicleID, Location: domain.Location{Lat: -6.2, Lon: 106.8, Timestamp: ts1}},
				{VehicleID: query.VehicleID, Location: domain.Location{Lat: -6.3, Lon: 106.9, Timestamp: ts2}},
			}, nil
		},
	}

	svc := NewLocationService(repo)
	query := &domain.HistoryQuery{
		VehicleID: "B1234XYZ",
		Start:     time.Unix(1715000000, 0),
		End:       time.Unix(1715009999, 0),
	}

	results, err := svc.GetHistory(context.Background(), query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestGetHistory_RepoError(t *testing.T) {
	repo := &mockLocationRepo{
		getHistoryFn: func(_ context.Context, _ *domain.HistoryQuery) ([]domain.VehicleLocation, error) {
			return nil, errors.New("db error")
		},
	}

	svc := NewLocationService(repo)
	_, err := svc.GetHistory(context.Background(), &domain.HistoryQuery{VehicleID: "X"})
	if err == nil {
		t.Fatal("expected error")
	}
}
