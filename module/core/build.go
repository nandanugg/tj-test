package core

import (
	"database/sql"
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/nandanugg/tj-test/module/core/domain"
	handler "github.com/nandanugg/tj-test/module/core/internal/handler/http"
	"github.com/nandanugg/tj-test/module/core/internal/handler/subscriber"
	"github.com/nandanugg/tj-test/module/core/internal/repository/database/postgres"
	"github.com/nandanugg/tj-test/module/core/internal/repository/publisher/rabbitmq"
	"github.com/nandanugg/tj-test/module/core/service"
)

type Module struct {
	LocationSvc *service.LocationService
	GeofenceSvc *service.GeofenceService
	handler     *handler.VehicleHandler
	subscriber  *subscriber.LocationSubscriber
}

func Build(db *sql.DB, amqpConn *amqp.Connection, mqttClient mqtt.Client, geofences []domain.GeoPoint) (*Module, error) {
	locationRepo := postgres.NewLocationRepo(db)

	geofencePub, err := rabbitmq.NewGeofencePublisher(amqpConn)
	if err != nil {
		return nil, fmt.Errorf("geofence publisher: %w", err)
	}

	locationSvc := service.NewLocationService(locationRepo)
	geofenceSvc := service.NewGeofenceService(geofencePub, geofences)

	h := handler.NewVehicleHandler(locationSvc)
	sub := subscriber.NewLocationSubscriber(mqttClient, locationSvc, geofenceSvc)

	return &Module{
		LocationSvc: locationSvc,
		GeofenceSvc: geofenceSvc,
		handler:     h,
		subscriber:  sub,
	}, nil
}

func (m *Module) RegisterRoutes(r *gin.RouterGroup) {
	m.handler.Register(r)
}

func (m *Module) StartSubscribers() error {
	return m.subscriber.Start()
}
