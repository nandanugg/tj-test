package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nandanugg/tj-test/module/core/domain"
)

type mockLocationService struct {
	getLatestFn      func(ctx context.Context, vehicleID string) (*domain.VehicleLocation, error)
	getHistoryFn     func(ctx context.Context, query *domain.HistoryQuery) ([]domain.VehicleLocation, error)
	getAllVehiclesFn func(ctx context.Context) ([]domain.Vehicle, error)
}

func (m *mockLocationService) GetLatest(ctx context.Context, vehicleID string) (*domain.VehicleLocation, error) {
	return m.getLatestFn(ctx, vehicleID)
}

func (m *mockLocationService) GetHistory(ctx context.Context, query *domain.HistoryQuery) ([]domain.VehicleLocation, error) {
	return m.getHistoryFn(ctx, query)
}

func (m *mockLocationService) GetAllVehicles(ctx context.Context) ([]domain.Vehicle, error) {
	return m.getAllVehiclesFn(ctx)
}

func setupRouter(svc locationService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewVehicleHandler(svc)
	h.Register(r.Group(""))
	return r
}

func TestGetLatestLocation_Success(t *testing.T) {
	ts := time.Unix(1715003456, 0)
	svc := &mockLocationService{
		getLatestFn: func(_ context.Context, vehicleID string) (*domain.VehicleLocation, error) {
			if vehicleID != "B1234XYZ" {
				t.Fatalf("unexpected vehicleID: %s", vehicleID)
			}
			return &domain.VehicleLocation{
				VehicleID: "B1234XYZ",
				Location:  domain.Location{Lat: -6.2088, Lon: 106.8456, Timestamp: ts},
			}, nil
		},
	}

	r := setupRouter(svc)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/vehicles/B1234XYZ/location", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp locationResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.VehicleID != "B1234XYZ" {
		t.Errorf("expected B1234XYZ, got %s", resp.VehicleID)
	}
	if resp.Latitude != -6.2088 {
		t.Errorf("expected -6.2088, got %f", resp.Latitude)
	}
	if resp.Timestamp != 1715003456 {
		t.Errorf("expected 1715003456, got %d", resp.Timestamp)
	}
}

func TestGetLatestLocation_NotFound(t *testing.T) {
	svc := &mockLocationService{
		getLatestFn: func(_ context.Context, _ string) (*domain.VehicleLocation, error) {
			return nil, errors.New("not found")
		},
	}

	r := setupRouter(svc)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/vehicles/UNKNOWN/location", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetHistory_Success(t *testing.T) {
	ts1 := time.Unix(1715000000, 0)
	ts2 := time.Unix(1715005000, 0)
	svc := &mockLocationService{
		getHistoryFn: func(_ context.Context, query *domain.HistoryQuery) ([]domain.VehicleLocation, error) {
			if query.VehicleID != "B1234XYZ" {
				t.Fatalf("unexpected vehicleID: %s", query.VehicleID)
			}
			return []domain.VehicleLocation{
				{VehicleID: "B1234XYZ", Location: domain.Location{Lat: -6.2, Lon: 106.8, Timestamp: ts1}},
				{VehicleID: "B1234XYZ", Location: domain.Location{Lat: -6.3, Lon: 106.9, Timestamp: ts2}},
			}, nil
		},
	}

	r := setupRouter(svc)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/vehicles/B1234XYZ/history?start=1715000000&end=1715009999", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp []locationResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp))
	}
}

func TestGetHistory_InvalidStart(t *testing.T) {
	svc := &mockLocationService{}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/vehicles/B1234XYZ/history?start=abc&end=1715009999", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetHistory_InvalidEnd(t *testing.T) {
	svc := &mockLocationService{}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/vehicles/B1234XYZ/history?start=1715000000&end=abc", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetHistory_ServiceError(t *testing.T) {
	svc := &mockLocationService{
		getHistoryFn: func(_ context.Context, _ *domain.HistoryQuery) ([]domain.VehicleLocation, error) {
			return nil, errors.New("db error")
		},
	}

	r := setupRouter(svc)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/vehicles/B1234XYZ/history?start=1715000000&end=1715009999", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestGetAllVehicles_Success(t *testing.T) {
	svc := &mockLocationService{
		getAllVehiclesFn: func(_ context.Context) ([]domain.Vehicle, error) {
			return []domain.Vehicle{
				{VehicleID: "B1234XYZ"},
				{VehicleID: "B5678ABC"},
			}, nil
		},
	}

	r := setupRouter(svc)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/vehicles", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp []domain.Vehicle
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("expected 2 vehicles, got %d", len(resp))
	}
	if resp[0].VehicleID != "B1234XYZ" {
		t.Errorf("expected B1234XYZ, got %s", resp[0].VehicleID)
	}
}

func TestGetAllVehicles_Error(t *testing.T) {
	svc := &mockLocationService{
		getAllVehiclesFn: func(_ context.Context) ([]domain.Vehicle, error) {
			return nil, errors.New("db error")
		},
	}

	r := setupRouter(svc)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/vehicles", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
