package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/nandanugg/tj-test/module/core/domain"
	"github.com/nandanugg/tj-test/module/core/internal/repository/publisher"
)

var _ publisher.GeofencePublisher = (*GeofencePublisher)(nil)

const (
	exchangeName = "fleet.events"
	queueName    = "geofence_alerts"
)

type GeofencePublisher struct {
	ch *amqp.Channel
}

func NewGeofencePublisher(conn *amqp.Connection) (*GeofencePublisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}

	if err := ch.ExchangeDeclare(exchangeName, "fanout", true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("declare exchange: %w", err)
	}

	if _, err := ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	if err := ch.QueueBind(queueName, "", exchangeName, false, nil); err != nil {
		return nil, fmt.Errorf("bind queue: %w", err)
	}

	return &GeofencePublisher{ch: ch}, nil
}

type alertMessage struct {
	VehicleID string                `json:"vehicle_id"`
	Event     domain.GeofenceEventType `json:"event"`
	Location  alertLocation         `json:"location"`
	Timestamp int64                 `json:"timestamp"`
}

type alertLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func (p *GeofencePublisher) PublishAlert(ctx context.Context, alert *domain.GeofenceAlert) error {
	msg := alertMessage{
		VehicleID: alert.VehicleID,
		Event:     alert.Event,
		Location: alertLocation{
			Latitude:  alert.Location.Lat,
			Longitude: alert.Location.Lon,
		},
		Timestamp: alert.Timestamp,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal alert: %w", err)
	}

	return p.ch.PublishWithContext(ctx, exchangeName, "", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}
