package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/nandanugg/tj-test/module/core/domain"
)

const topicPattern = "/fleet/vehicle/+/location"

type locationService interface {
	SaveLocation(ctx context.Context, vl *domain.VehicleLocation) error
}

type geofenceService interface {
	CheckAndAlert(ctx context.Context, vl *domain.VehicleLocation) error
}

type locationMessage struct {
	VehicleID string  `json:"vehicle_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp int64   `json:"timestamp"`
}

type LocationSubscriber struct {
	client      mqtt.Client
	locationSvc locationService
	geofenceSvc geofenceService
}

func NewLocationSubscriber(client mqtt.Client, locationSvc locationService, geofenceSvc geofenceService) *LocationSubscriber {
	return &LocationSubscriber{
		client:      client,
		locationSvc: locationSvc,
		geofenceSvc: geofenceSvc,
	}
}

func (s *LocationSubscriber) Start() error {
	token := s.client.Subscribe(topicPattern, 1, s.handleMessage)
	token.Wait()
	return token.Error()
}

func (s *LocationSubscriber) handleMessage(_ mqtt.Client, msg mqtt.Message) {
	var raw locationMessage
	if err := json.Unmarshal(msg.Payload(), &raw); err != nil {
		log.Printf("invalid location message: %v", err)
		return
	}

	if err := validateLocationMessage(&raw); err != nil {
		log.Printf("validation error: %v", err)
		return
	}

	vl := &domain.VehicleLocation{
		VehicleID: raw.VehicleID,
		Location: domain.Location{
			Lat:       raw.Latitude,
			Lon:       raw.Longitude,
			Timestamp: time.Unix(raw.Timestamp, 0),
		},
	}

	ctx := context.Background()

	if err := s.locationSvc.SaveLocation(ctx, vl); err != nil {
		log.Printf("save location error: %v", err)
		return
	}

	if err := s.geofenceSvc.CheckAndAlert(ctx, vl); err != nil {
		log.Printf("geofence check error: %v", err)
	}
}

func validateLocationMessage(msg *locationMessage) error {
	if msg.VehicleID == "" {
		return fmt.Errorf("vehicle_id: required")
	}
	if msg.Latitude < -90 || msg.Latitude > 90 {
		return fmt.Errorf("latitude: must be between -90 and 90")
	}
	if msg.Longitude < -180 || msg.Longitude > 180 {
		return fmt.Errorf("longitude: must be between -180 and 180")
	}
	if msg.Timestamp <= 0 {
		return fmt.Errorf("timestamp: must be positive")
	}
	return nil
}
