package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/nandanugg/tj-test/config"
	"github.com/nandanugg/tj-test/module/core"
	"github.com/nandanugg/tj-test/module/core/domain"
)

func main() {
	cfg := config.Load()

	db, err := config.NewPostgres(cfg)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer func() { _ = db.Close() }()

	amqpConn, err := config.NewRabbitMQ(cfg)
	if err != nil {
		log.Fatalf("rabbitmq: %v", err)
	}
	defer func() { _ = amqpConn.Close() }()

	mqttClient, err := config.NewMQTT(cfg)
	if err != nil {
		log.Fatalf("mqtt: %v", err)
	}
	defer mqttClient.Disconnect(250)

	geofences := []domain.GeoPoint{
		{Lat: -6.2088, Lon: 106.8456, Radius: 50},
	}

	coreModule, err := core.Build(db, amqpConn, mqttClient, geofences)
	if err != nil {
		log.Fatalf("core module: %v", err)
	}

	if err := coreModule.StartSubscribers(); err != nil {
		log.Fatalf("start subscribers: %v", err)
	}

	r := gin.Default()

	health := config.NewHealthChecker(db, amqpConn, mqttClient)
	health.Register(r)

	coreModule.RegisterRoutes(&r.RouterGroup)

	log.Printf("listening on :%s", cfg.HTTPPort)
	if err := r.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatalf("server: %v", err)
	}
}
