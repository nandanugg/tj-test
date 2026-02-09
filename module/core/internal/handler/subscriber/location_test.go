package subscriber

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/nandanugg/tj-test/module/core/domain"
)

type mockLocationSvc struct {
	saveLocationFn func(ctx context.Context, vl *domain.VehicleLocation) error
}

func (m *mockLocationSvc) SaveLocation(ctx context.Context, vl *domain.VehicleLocation) error {
	return m.saveLocationFn(ctx, vl)
}

type mockGeofenceSvc struct {
	checkAndAlertFn func(ctx context.Context, vl *domain.VehicleLocation) error
}

func (m *mockGeofenceSvc) CheckAndAlert(ctx context.Context, vl *domain.VehicleLocation) error {
	return m.checkAndAlertFn(ctx, vl)
}

type fakeMQTTMessage struct {
	payload []byte
}

func (f *fakeMQTTMessage) Duplicate() bool   { return false }
func (f *fakeMQTTMessage) Qos() byte         { return 0 }
func (f *fakeMQTTMessage) Retained() bool    { return false }
func (f *fakeMQTTMessage) Topic() string     { return "/fleet/vehicle/B1234XYZ/location" }
func (f *fakeMQTTMessage) MessageID() uint16 { return 0 }
func (f *fakeMQTTMessage) Payload() []byte   { return f.payload }
func (f *fakeMQTTMessage) Ack()              {}

func TestHandleMessage_Success(t *testing.T) {
	var savedVL *domain.VehicleLocation
	var checkedVL *domain.VehicleLocation

	locSvc := &mockLocationSvc{
		saveLocationFn: func(_ context.Context, vl *domain.VehicleLocation) error {
			savedVL = vl
			return nil
		},
	}
	geoSvc := &mockGeofenceSvc{
		checkAndAlertFn: func(_ context.Context, vl *domain.VehicleLocation) error {
			checkedVL = vl
			return nil
		},
	}

	sub := &LocationSubscriber{locationSvc: locSvc, geofenceSvc: geoSvc}

	msg := locationMessage{
		VehicleID: "B1234XYZ",
		Latitude:  -6.2088,
		Longitude: 106.8456,
		Timestamp: 1715003456,
	}
	payload, _ := json.Marshal(msg)
	sub.handleMessage(nil, &fakeMQTTMessage{payload: payload})

	if savedVL == nil {
		t.Fatal("expected SaveLocation to be called")
	}
	if savedVL.VehicleID != "B1234XYZ" {
		t.Errorf("expected B1234XYZ, got %s", savedVL.VehicleID)
	}
	if savedVL.Location.Lat != -6.2088 {
		t.Errorf("expected -6.2088, got %f", savedVL.Location.Lat)
	}
	expectedTs := time.Unix(1715003456, 0)
	if !savedVL.Location.Timestamp.Equal(expectedTs) {
		t.Errorf("expected %v, got %v", expectedTs, savedVL.Location.Timestamp)
	}
	if checkedVL == nil {
		t.Fatal("expected CheckAndAlert to be called")
	}
}

func TestHandleMessage_InvalidJSON(t *testing.T) {
	locSvc := &mockLocationSvc{
		saveLocationFn: func(_ context.Context, _ *domain.VehicleLocation) error {
			t.Fatal("SaveLocation should not be called")
			return nil
		},
	}
	geoSvc := &mockGeofenceSvc{}

	sub := &LocationSubscriber{locationSvc: locSvc, geofenceSvc: geoSvc}
	sub.handleMessage(nil, &fakeMQTTMessage{payload: []byte("invalid")})
}

func TestHandleMessage_ValidationError(t *testing.T) {
	locSvc := &mockLocationSvc{
		saveLocationFn: func(_ context.Context, _ *domain.VehicleLocation) error {
			t.Fatal("SaveLocation should not be called")
			return nil
		},
	}
	geoSvc := &mockGeofenceSvc{}

	sub := &LocationSubscriber{locationSvc: locSvc, geofenceSvc: geoSvc}

	// empty vehicle_id
	msg := locationMessage{Latitude: -6.2, Longitude: 106.8, Timestamp: 1715003456}
	payload, _ := json.Marshal(msg)
	sub.handleMessage(nil, &fakeMQTTMessage{payload: payload})
}

func TestHandleMessage_SaveError_SkipsGeofence(t *testing.T) {
	locSvc := &mockLocationSvc{
		saveLocationFn: func(_ context.Context, _ *domain.VehicleLocation) error {
			return errors.New("db error")
		},
	}
	geoSvc := &mockGeofenceSvc{
		checkAndAlertFn: func(_ context.Context, _ *domain.VehicleLocation) error {
			t.Fatal("CheckAndAlert should not be called when save fails")
			return nil
		},
	}

	sub := &LocationSubscriber{locationSvc: locSvc, geofenceSvc: geoSvc}

	msg := locationMessage{VehicleID: "B1234XYZ", Latitude: -6.2, Longitude: 106.8, Timestamp: 1715003456}
	payload, _ := json.Marshal(msg)
	sub.handleMessage(nil, &fakeMQTTMessage{payload: payload})
}

func TestValidateLocationMessage(t *testing.T) {
	tests := []struct {
		name    string
		msg     locationMessage
		wantErr bool
	}{
		{"valid", locationMessage{VehicleID: "X", Latitude: 0, Longitude: 0, Timestamp: 1}, false},
		{"empty vehicle_id", locationMessage{Latitude: 0, Longitude: 0, Timestamp: 1}, true},
		{"lat too low", locationMessage{VehicleID: "X", Latitude: -91, Longitude: 0, Timestamp: 1}, true},
		{"lat too high", locationMessage{VehicleID: "X", Latitude: 91, Longitude: 0, Timestamp: 1}, true},
		{"lon too low", locationMessage{VehicleID: "X", Latitude: 0, Longitude: -181, Timestamp: 1}, true},
		{"lon too high", locationMessage{VehicleID: "X", Latitude: 0, Longitude: 181, Timestamp: 1}, true},
		{"zero timestamp", locationMessage{VehicleID: "X", Latitude: 0, Longitude: 0, Timestamp: 0}, true},
		{"negative timestamp", locationMessage{VehicleID: "X", Latitude: 0, Longitude: 0, Timestamp: -1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLocationMessage(&tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateLocationMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
