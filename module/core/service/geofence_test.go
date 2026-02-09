package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nandanugg/tj-test/module/core/domain"
)

type mockGeofencePublisher struct {
	publishAlertFn func(ctx context.Context, alert *domain.GeofenceAlert) error
	calls          []*domain.GeofenceAlert
}

func (m *mockGeofencePublisher) PublishAlert(ctx context.Context, alert *domain.GeofenceAlert) error {
	m.calls = append(m.calls, alert)
	if m.publishAlertFn != nil {
		return m.publishAlertFn(ctx, alert)
	}
	return nil
}

func TestCheckAndAlert_InsideGeofence(t *testing.T) {
	pub := &mockGeofencePublisher{}
	geofences := []domain.GeoPoint{
		{Lat: -6.2088, Lon: 106.8456, Radius: 50},
	}
	svc := NewGeofenceService(pub, geofences)

	// exact same point — distance is 0, within 50m
	vl := &domain.VehicleLocation{
		VehicleID: "B1234XYZ",
		Location: domain.Location{
			Lat:       -6.2088,
			Lon:       106.8456,
			Timestamp: time.Unix(1715003456, 0),
		},
	}

	err := svc.CheckAndAlert(context.Background(), vl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pub.calls) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(pub.calls))
	}
	alert := pub.calls[0]
	if alert.VehicleID != "B1234XYZ" {
		t.Errorf("expected B1234XYZ, got %s", alert.VehicleID)
	}
	if alert.Event != domain.GeofenceEntry {
		t.Errorf("expected geofence_entry, got %s", alert.Event)
	}
}

func TestCheckAndAlert_OutsideGeofence(t *testing.T) {
	pub := &mockGeofencePublisher{}
	geofences := []domain.GeoPoint{
		{Lat: -6.2088, Lon: 106.8456, Radius: 50},
	}
	svc := NewGeofenceService(pub, geofences)

	// far away point
	vl := &domain.VehicleLocation{
		VehicleID: "B1234XYZ",
		Location: domain.Location{
			Lat:       -7.0,
			Lon:       107.0,
			Timestamp: time.Unix(1715003456, 0),
		},
	}

	err := svc.CheckAndAlert(context.Background(), vl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pub.calls) != 0 {
		t.Fatalf("expected 0 alerts, got %d", len(pub.calls))
	}
}

func TestCheckAndAlert_MultipleGeofences(t *testing.T) {
	pub := &mockGeofencePublisher{}
	geofences := []domain.GeoPoint{
		{Lat: -6.2088, Lon: 106.8456, Radius: 50},
		{Lat: -6.2088, Lon: 106.8456, Radius: 100}, // overlapping
		{Lat: -7.0, Lon: 107.0, Radius: 50},        // far away
	}
	svc := NewGeofenceService(pub, geofences)

	vl := &domain.VehicleLocation{
		VehicleID: "B1234XYZ",
		Location: domain.Location{
			Lat:       -6.2088,
			Lon:       106.8456,
			Timestamp: time.Unix(1715003456, 0),
		},
	}

	err := svc.CheckAndAlert(context.Background(), vl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pub.calls) != 2 {
		t.Fatalf("expected 2 alerts, got %d", len(pub.calls))
	}
}

func TestCheckAndAlert_PublishError(t *testing.T) {
	pub := &mockGeofencePublisher{
		publishAlertFn: func(_ context.Context, _ *domain.GeofenceAlert) error {
			return errors.New("rabbitmq down")
		},
	}
	geofences := []domain.GeoPoint{
		{Lat: -6.2088, Lon: 106.8456, Radius: 50},
	}
	svc := NewGeofenceService(pub, geofences)

	vl := &domain.VehicleLocation{
		VehicleID: "B1234XYZ",
		Location: domain.Location{
			Lat:       -6.2088,
			Lon:       106.8456,
			Timestamp: time.Unix(1715003456, 0),
		},
	}

	err := svc.CheckAndAlert(context.Background(), vl)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCheckAndAlert_NoGeofences(t *testing.T) {
	pub := &mockGeofencePublisher{}
	svc := NewGeofenceService(pub, nil)

	vl := &domain.VehicleLocation{
		VehicleID: "B1234XYZ",
		Location: domain.Location{
			Lat:       -6.2088,
			Lon:       106.8456,
			Timestamp: time.Unix(1715003456, 0),
		},
	}

	err := svc.CheckAndAlert(context.Background(), vl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pub.calls) != 0 {
		t.Fatalf("expected 0 alerts, got %d", len(pub.calls))
	}
}

func TestHaversine(t *testing.T) {
	// same point should be 0
	d := haversine(-6.2088, 106.8456, -6.2088, 106.8456)
	if d != 0 {
		t.Errorf("expected 0, got %f", d)
	}

	// roughly 157m between these two points
	d = haversine(-6.2088, 106.8456, -6.2100, 106.8456)
	if d < 100 || d > 200 {
		t.Errorf("expected ~133m, got %f", d)
	}
}
